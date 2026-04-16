package inframetrics

import (
	"context"
	"math"
	"sort"

	"github.com/SigNoz/signoz/pkg/query-service/app/metrics/v4/helpers"
	"github.com/SigNoz/signoz/pkg/query-service/interfaces"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	v3 "github.com/SigNoz/signoz/pkg/query-service/model/v3"
	"github.com/SigNoz/signoz/pkg/query-service/postprocess"
	"github.com/SigNoz/signoz/pkg/valuer"
	"golang.org/x/exp/slices"
)

var (
	metricToUseForRollouts = GetDotMetrics("k8s_pod_cpu_usage")
	k8sRolloutNameAttrKey  = GetDotMetrics("k8s_rollout_name")

	rolloutAttrsToEnrich = []string{
		GetDotMetrics("k8s_rollout_name"),
		GetDotMetrics("k8s_namespace_name"),
		GetDotMetrics("k8s_cluster_name"),
	}

	queryNamesForRollouts = map[string][]string{
		"cpu":            {"A"},
		"cpu_request":    {"B", "A"},
		"cpu_limit":      {"C", "A"},
		"memory":         {"D"},
		"memory_request": {"E", "D"},
		"memory_limit":   {"F", "D"},
		"restarts":       {"G", "A"},
	}

	rolloutQueryNames = []string{"A", "B", "C", "D", "E", "F", "G"}
)

type RolloutsRepo struct {
	reader    interfaces.Reader
	querierV2 interfaces.Querier
}

func NewRolloutsRepo(reader interfaces.Reader, querierV2 interfaces.Querier) *RolloutsRepo {
	return &RolloutsRepo{reader: reader, querierV2: querierV2}
}

func (r *RolloutsRepo) GetRolloutAttributeKeys(ctx context.Context, req v3.FilterAttributeKeyRequest) (*v3.FilterAttributeKeyResponse, error) {
	req.DataSource = v3.DataSourceMetrics
	req.AggregateAttribute = metricToUseForRollouts
	if req.Limit == 0 {
		req.Limit = 50
	}

	attributeKeysResponse, err := r.reader.GetMetricAttributeKeys(ctx, &req)
	if err != nil {
		return nil, err
	}

	filteredKeys := []v3.AttributeKey{}
	for _, key := range attributeKeysResponse.AttributeKeys {
		if slices.Contains(pointAttrsToIgnore, key.Key) {
			continue
		}
		filteredKeys = append(filteredKeys, key)
	}

	return &v3.FilterAttributeKeyResponse{AttributeKeys: filteredKeys}, nil
}

func (r *RolloutsRepo) GetRolloutAttributeValues(ctx context.Context, req v3.FilterAttributeValueRequest) (*v3.FilterAttributeValueResponse, error) {
	req.DataSource = v3.DataSourceMetrics
	req.AggregateAttribute = metricToUseForRollouts
	if req.Limit == 0 {
		req.Limit = 50
	}

	attributeValuesResponse, err := r.reader.GetMetricAttributeValues(ctx, &req)
	if err != nil {
		return nil, err
	}

	return attributeValuesResponse, nil
}

func (r *RolloutsRepo) getMetadataAttributes(ctx context.Context, req model.RolloutListRequest) (map[string]map[string]string, error) {
	rolloutAttrs := map[string]map[string]string{}

	for _, key := range rolloutAttrsToEnrich {
		hasKey := false
		for _, groupByKey := range req.GroupBy {
			if groupByKey.Key == key {
				hasKey = true
				break
			}
		}
		if !hasKey {
			req.GroupBy = append(req.GroupBy, v3.AttributeKey{Key: key})
		}
	}

	mq := v3.BuilderQuery{
		DataSource: v3.DataSourceMetrics,
		AggregateAttribute: v3.AttributeKey{
			Key:      metricToUseForRollouts,
			DataType: v3.AttributeKeyDataTypeFloat64,
		},
		Temporality: v3.Unspecified,
		GroupBy:     req.GroupBy,
	}

	query, err := helpers.PrepareTimeseriesFilterQuery(req.Start, req.End, &mq)
	if err != nil {
		return nil, err
	}

	query = localQueryToDistributedQuery(query)

	attrsListResponse, err := r.reader.GetListResultV3(ctx, query)
	if err != nil {
		return nil, err
	}

	for _, row := range attrsListResponse {
		stringData := map[string]string{}
		for key, value := range row.Data {
			if str, ok := value.(string); ok {
				stringData[key] = str
			} else if strPtr, ok := value.(*string); ok {
				stringData[key] = *strPtr
			}
		}

		rolloutName := stringData[k8sRolloutNameAttrKey]
		namespaceName := stringData[k8sNamespaceNameAttrKey]
		clusterName := stringData[k8sClusterNameAttrKey]
		metaKey := workloadMetaKey(rolloutName, namespaceName, clusterName)
		if _, ok := rolloutAttrs[metaKey]; !ok {
			rolloutAttrs[metaKey] = map[string]string{}
		}

		for _, key := range req.GroupBy {
			rolloutAttrs[metaKey][key.Key] = stringData[key.Key]
		}
	}

	return rolloutAttrs, nil
}

func (r *RolloutsRepo) getTopRolloutGroups(ctx context.Context, orgID valuer.UUID, req model.RolloutListRequest, q *v3.QueryRangeParamsV3) ([]map[string]string, []map[string]string, error) {
	step, timeSeriesTableName, samplesTableName := getParamsForTopRollouts(req)

	queryNames := queryNamesForRollouts[req.OrderBy.ColumnName]
	topRolloutGroupsQueryRangeParams := &v3.QueryRangeParamsV3{
		Start: req.Start,
		End:   req.End,
		Step:  step,
		CompositeQuery: &v3.CompositeQuery{
			BuilderQueries: map[string]*v3.BuilderQuery{},
			QueryType:      v3.QueryTypeBuilder,
			PanelType:      v3.PanelTypeTable,
		},
	}

	for _, queryName := range queryNames {
		query := q.CompositeQuery.BuilderQueries[queryName].Clone()
		query.StepInterval = step
		query.MetricTableHints = &v3.MetricTableHints{
			TimeSeriesTableName: timeSeriesTableName,
			SamplesTableName:    samplesTableName,
		}
		if req.Filters != nil && len(req.Filters.Items) > 0 {
			if query.Filters == nil {
				query.Filters = &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{}}
			}
			query.Filters.Items = append(query.Filters.Items, req.Filters.Items...)
		}
		topRolloutGroupsQueryRangeParams.CompositeQuery.BuilderQueries[queryName] = query
	}

	queryResponse, _, err := r.querierV2.QueryRange(ctx, orgID, topRolloutGroupsQueryRangeParams)
	if err != nil {
		return nil, nil, err
	}
	formattedResponse, err := postprocess.PostProcessResult(queryResponse, topRolloutGroupsQueryRangeParams)
	if err != nil {
		return nil, nil, err
	}

	if len(formattedResponse) == 0 || len(formattedResponse[0].Series) == 0 {
		return nil, nil, nil
	}

	if req.OrderBy.Order == v3.DirectionDesc {
		sort.Slice(formattedResponse[0].Series, func(i, j int) bool {
			return formattedResponse[0].Series[i].Points[0].Value > formattedResponse[0].Series[j].Points[0].Value
		})
	} else {
		sort.Slice(formattedResponse[0].Series, func(i, j int) bool {
			return formattedResponse[0].Series[i].Points[0].Value < formattedResponse[0].Series[j].Points[0].Value
		})
	}

	limit := math.Min(float64(req.Offset+req.Limit), float64(len(formattedResponse[0].Series)))

	paginatedTopRolloutGroupsSeries := formattedResponse[0].Series[req.Offset:int(limit)]

	topRolloutGroups := []map[string]string{}
	for _, series := range paginatedTopRolloutGroupsSeries {
		topRolloutGroups = append(topRolloutGroups, series.Labels)
	}
	allRolloutGroups := []map[string]string{}
	for _, series := range formattedResponse[0].Series {
		allRolloutGroups = append(allRolloutGroups, series.Labels)
	}

	return topRolloutGroups, allRolloutGroups, nil
}

func (r *RolloutsRepo) GetRolloutList(ctx context.Context, orgID valuer.UUID, req model.RolloutListRequest) (model.RolloutListResponse, error) {
	resp := model.RolloutListResponse{}

	if req.Limit == 0 {
		req.Limit = 10
	}

	if req.OrderBy == nil {
		req.OrderBy = &v3.OrderBy{ColumnName: "cpu", Order: v3.DirectionDesc}
	}

	if req.GroupBy == nil {
		req.GroupBy = []v3.AttributeKey{
			{Key: k8sRolloutNameAttrKey},
			{Key: k8sNamespaceNameAttrKey},
			{Key: k8sClusterNameAttrKey},
		}
		resp.Type = model.ResponseTypeList
	} else {
		resp.Type = model.ResponseTypeGroupedList
	}

	step, timeSeriesTableName, samplesTableName := getParamsForTopRollouts(req)

	query := WorkloadTableListQuery.Clone()

	query.Start = req.Start
	query.End = req.End
	query.Step = step

	// No additional builder queries for rollouts (no H/I like deployments)
	// Rollouts use only the common A-G workload queries

	for _, query := range query.CompositeQuery.BuilderQueries {
		query.StepInterval = step
		query.MetricTableHints = &v3.MetricTableHints{
			TimeSeriesTableName: timeSeriesTableName,
			SamplesTableName:    samplesTableName,
		}
		if req.Filters != nil && len(req.Filters.Items) > 0 {
			if query.Filters == nil {
				query.Filters = &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{}}
			}
			query.Filters.Items = append(query.Filters.Items, req.Filters.Items...)
		}
		query.GroupBy = req.GroupBy
		// make sure we only get records for rollouts
		query.Filters.Items = append(query.Filters.Items, v3.FilterItem{
			Key:      v3.AttributeKey{Key: k8sRolloutNameAttrKey},
			Operator: v3.FilterOperatorExists,
		})
	}

	// Run metadata and top groups queries in parallel
	var rolloutAttrs map[string]map[string]string
	var topRolloutGroups, allRolloutGroups []map[string]string
	var metaErr, topErr error

	done := make(chan struct{})
	go func() {
		rolloutAttrs, metaErr = r.getMetadataAttributes(ctx, req)
		done <- struct{}{}
	}()
	topRolloutGroups, allRolloutGroups, topErr = r.getTopRolloutGroups(ctx, orgID, req, query)
	<-done

	if metaErr != nil {
		return resp, metaErr
	}
	if topErr != nil {
		return resp, topErr
	}

	groupFilters := map[string][]string{}
	for _, topRolloutGroup := range topRolloutGroups {
		for k, v := range topRolloutGroup {
			groupFilters[k] = append(groupFilters[k], v)
		}
	}

	for groupKey, groupValues := range groupFilters {
		hasGroupFilter := false
		if req.Filters != nil && len(req.Filters.Items) > 0 {
			for _, filter := range req.Filters.Items {
				if filter.Key.Key == groupKey {
					hasGroupFilter = true
					break
				}
			}
		}

		if !hasGroupFilter {
			for _, query := range query.CompositeQuery.BuilderQueries {
				query.Filters.Items = append(query.Filters.Items, v3.FilterItem{
					Key:      v3.AttributeKey{Key: groupKey},
					Value:    groupValues,
					Operator: v3.FilterOperatorIn,
				})
			}
		}
	}

	queryResponse, _, err := r.querierV2.QueryRange(ctx, orgID, query)
	if err != nil {
		return resp, err
	}

	formattedResponse, err := postprocess.PostProcessResult(queryResponse, query)
	if err != nil {
		return resp, err
	}

	records := []model.RolloutListRecord{}

	for _, result := range formattedResponse {
		for _, row := range result.Table.Rows {

			record := model.RolloutListRecord{
				RolloutName:   "",
				CPUUsage:      -1,
				CPURequest:    -1,
				CPULimit:      -1,
				MemoryUsage:   -1,
				MemoryRequest: -1,
				MemoryLimit:   -1,
			}

			if rolloutName, ok := row.Data[k8sRolloutNameAttrKey].(string); ok {
				record.RolloutName = rolloutName
			}

			if cpu, ok := row.Data["A"].(float64); ok {
				record.CPUUsage = cpu
			}
			if cpuRequest, ok := row.Data["B"].(float64); ok {
				record.CPURequest = cpuRequest
			}
			if cpuLimit, ok := row.Data["C"].(float64); ok {
				record.CPULimit = cpuLimit
			}
			if memory, ok := row.Data["D"].(float64); ok {
				record.MemoryUsage = memory
			}
			if memoryRequest, ok := row.Data["E"].(float64); ok {
				record.MemoryRequest = memoryRequest
			}
			if memoryLimit, ok := row.Data["F"].(float64); ok {
				record.MemoryLimit = memoryLimit
			}
			if restarts, ok := row.Data["G"].(float64); ok {
				record.Restarts = int(restarts)
			}

			record.Meta = map[string]string{}
			namespaceName := ""
			if ns, ok := row.Data[k8sNamespaceNameAttrKey].(string); ok {
				namespaceName = ns
			}
			clusterName := ""
			if cl, ok := row.Data[k8sClusterNameAttrKey].(string); ok {
				clusterName = cl
			}
			metaKey := workloadMetaKey(record.RolloutName, namespaceName, clusterName)
			if _, ok := rolloutAttrs[metaKey]; ok && record.RolloutName != "" {
				record.Meta = rolloutAttrs[metaKey]
			}

			for k, v := range row.Data {
				if slices.Contains(rolloutQueryNames, k) {
					continue
				}
				if labelValue, ok := v.(string); ok {
					record.Meta[k] = labelValue
				}
			}

			records = append(records, record)
		}
	}

	// Post-filter: eliminate cross-product false matches.
	// Independent IN filters (name IN [...] AND namespace IN [...]) create a superset;
	// only keep records matching exact top group tuples.
	if len(topRolloutGroups) > 0 {
		groupByKeys := make([]string, len(req.GroupBy))
		for i, gb := range req.GroupBy {
			groupByKeys[i] = gb.Key
		}
		validGroups := make(map[string]bool, len(topRolloutGroups))
		for _, g := range topRolloutGroups {
			validGroups[buildGroupTupleKey(g, groupByKeys)] = true
		}
		filteredRecords := make([]model.RolloutListRecord, 0, len(topRolloutGroups))
		for _, record := range records {
			if validGroups[buildGroupTupleKey(record.Meta, groupByKeys)] {
				filteredRecords = append(filteredRecords, record)
			}
		}
		records = filteredRecords
	}

	resp.Total = len(allRolloutGroups)
	resp.Records = records

	resp.SortBy(req.OrderBy)

	return resp, nil
}
