import React, { memo, useMemo, useState, useEffect, useRef, useCallback } from 'react';
import { NetworkEdgeData, EdgeTraceDetailResponse, TimeRange } from '../types';
import './NetworkEdgeDetailPanel.css';
import { nextIcon, filterLightmode, searchSvg, closeWindow } from '../../../assets/ServiceMapIcons';
import { useTranslation } from 'react-i18next';
import { Input } from 'antd';
import { getEdgeTraceDetails } from 'api/servicemap';
import NetworkMapLoader from './NetworkMapLoader';
import KO11yTable from './KO11yTable';
import { useTimezone } from '../../../providers/Timezone';
import useTimezoneFormatter from '../../../hooks/useTimezoneFormatter/useTimezoneFormatter';

const formatLatency = (latency: number): string => {
  if (latency >= 1000) return `${(latency / 1000).toFixed(2)} s`;
  else if (latency >= 1) return `${latency.toFixed(2)} ms`;
  else return `${(latency * 1000).toFixed(2)} μs`;
};

// Path를 표시하는 컴포넌트 (복사 기능 제거)
const PathCell: React.FC<{ path: string }> = ({ path }) => {
  return (
    <div className="path-cell" title={path}>
      {path}
    </div>
  );
};

// TraceId를 표시하고 클릭하면 우측 Trace 상세 패널을 여는 컴포넌트
const TraceIdCell: React.FC<{ traceId: string; onClick?: (traceId: string) => void }> = ({ traceId, onClick }) => {
  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    onClick?.(traceId);
  };

  return (
    <button
      type="button"
      className="trace-id-cell"
      onClick={handleClick}
      title="Open trace details"
    >
      {traceId}
    </button>
  );
};

interface NetworkEdgeDetailPanelProps {
  edgeData: NetworkEdgeData;
  onClose: () => void;
  onTraceClick?: (traceId: string) => void;
  timeRange: TimeRange;
}

const NetworkEdgeDetailPanel: React.FC<NetworkEdgeDetailPanelProps> = memo(({
  edgeData,
  onClose,
  onTraceClick,
  timeRange
}) => {
  const [searchText, setSearchText] = useState('');
  const [showFilterDropdown, setShowFilterDropdown] = useState(false);
  const [selectedFilter, setSelectedFilter] = useState('');
  const [traceData, setTraceData] = useState<EdgeTraceDetailResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [allRequests, setAllRequests] = useState<any[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [collapsedSections, setCollapsedSections] = useState<Record<string, boolean>>({});
  const PAGE_SIZE = 20;
  const { t } = useTranslation('network_map');
  const traceSectionRef = useRef<HTMLDivElement>(null);
  const [allReqTableHeight, setAllReqTableHeight] = useState(300);
  const { timezone } = useTimezone();
  const { formatTimezoneAdjustedTimestamp } = useTimezoneFormatter({ userTimezone: timezone });

  const calcTableHeight = useCallback(() => {
    if (!traceSectionRef.current) return;
    const sectionRect = traceSectionRef.current.getBoundingClientRect();
    const header = traceSectionRef.current.querySelector('.section-header');
    const pagination = traceSectionRef.current.querySelector('.pagination-container');
    const headerH = header ? header.getBoundingClientRect().height : 0;
    const paginationH = pagination ? pagination.getBoundingClientRect().height : 0;
    // section-header 높이 + pagination 높이 + ant table header(~40px) 제외
    const available = sectionRect.height - headerH - paginationH - 40;
    if (available > 100) setAllReqTableHeight(Math.floor(available));
  }, []);

  useEffect(() => {
    const timer = setTimeout(calcTableHeight, 100);
    window.addEventListener('resize', calcTableHeight);
    return () => {
      clearTimeout(timer);
      window.removeEventListener('resize', calcTableHeight);
    };
  }, [calcTableHeight]);

  useEffect(() => {
    // 데이터 로딩 완료 후 높이 재계산
    if (traceData) setTimeout(calcTableHeight, 50);
  }, [traceData, calcTableHeight]);


  useEffect(() => {
    const fetchTraceData = async () => {
      if (!edgeData.id || !edgeData.source || !edgeData.destination) return;

      setIsLoading(true);
      setAllRequests([]);
      try {
        // 전역 timeRange 사용
        const endTime = timeRange.end;
        const startTime = timeRange.start;

        // source와 destination 파싱: "cluster$$namespace$$workload" 형식
        const parseEdgeId = (edgeId: string): { cluster: string; namespace: string; workload: string } => {
          const parts = edgeId.split('$$');
          return {
            cluster: parts[0] || '',
            namespace: parts[1] || '',
            workload: parts[2] || ''
          };
        };

        const sourceInfo = parseEdgeId(edgeData.source);
        const destInfo = parseEdgeId(edgeData.destination);

        // cluster와 namespace가 모두 "external"이면 외부 서비스
        const isClientExternal = (sourceInfo.cluster === 'external' && sourceInfo.namespace === 'external') ? 1 : 0;
        const isServerExternal = (destInfo.cluster === 'external' && destInfo.namespace === 'external') ? 1 : 0;

        const requestParams = {
          edgeId: edgeData.id,
          source: edgeData.source,
          destination: edgeData.destination,
          sourceRaw: edgeData.srcRaw || '',       // ✅ 원본 src 값 (trace 매칭용)
          destinationRaw: edgeData.destRaw || '', // ✅ 원본 dest 값 (trace 매칭용)
          startTime,
          endTime,
          isClientExternal,
          isServerExternal
        };

        console.log('[EdgeDetailPanel] Request params:', requestParams);

        const response = await getEdgeTraceDetails(requestParams);

        setTraceData(response);
        if (response?.requests) {
          setAllRequests(response.requests);
        }
      } catch (error) {
        console.error('Failed to fetch trace data:', error);
        setTraceData(null);
      } finally {
        setIsLoading(false);
      }
    };

    fetchTraceData();
  }, [edgeData.id, timeRange.start, timeRange.end]);


  const connectionInfo = useMemo((): { sourceWorkload: string; targetWorkload: string; protocol: string } => {
    return {
      sourceWorkload: traceData?.srcWorkload || t('loading') || 'Loading...',
      targetWorkload: traceData?.destWorkload || t('loading') || 'Loading...',
      protocol: traceData?.protocol || t('loading') || 'Loading...'
    };
  }, [traceData, t]);

  const filteredData = useMemo(() => {
    const requestsToFilter = allRequests.map(req => ({
      time: formatTimezoneAdjustedTimestamp(req.timestamp),
      traceId: req.traceId,
      method: req.method,
      path: req.path,
      status: req.status,
      isError: req.isError,
      latency: req.latency
    }));
    let filtered = [...requestsToFilter];

    if (searchText) {
      if (selectedFilter) {
        switch (selectedFilter) {
          case 'traceId':
            filtered = filtered.filter(trace => 
              trace.traceId.toLowerCase().includes(searchText.toLowerCase())
            );
            break;
          case 'workload':
            filtered = filtered.filter(trace => 
              connectionInfo.sourceWorkload.toLowerCase().includes(searchText.toLowerCase())
            );
            break;
          case 'method':
            filtered = filtered.filter(trace => 
              trace.method.toLowerCase().includes(searchText.toLowerCase())
            );
            break;
          case 'status':
            filtered = filtered.filter(trace => 
              trace.status.toString().includes(searchText)
            );
            break;
        }
      } else {
        filtered = filtered.filter(trace => 
          trace.time.toLowerCase().includes(searchText.toLowerCase()) ||
          trace.traceId.toLowerCase().includes(searchText.toLowerCase()) ||
          trace.method.toLowerCase().includes(searchText.toLowerCase()) ||
          trace.path.toLowerCase().includes(searchText.toLowerCase()) ||
          connectionInfo.sourceWorkload.toLowerCase().includes(searchText.toLowerCase()) ||
          trace.status.toString().includes(searchText)
        );
      }
    }

    return filtered;
  }, [allRequests, searchText, selectedFilter, connectionInfo.sourceWorkload, formatTimezoneAdjustedTimestamp]);

  // 검색/필터 변경 시 1페이지로 리셋
  useEffect(() => {
    setCurrentPage(1);
  }, [searchText, selectedFilter]);

  // 페이지네이션 계산
  const totalItems = filteredData.length;
  const totalPages = Math.max(1, Math.ceil(totalItems / PAGE_SIZE));
  const startIndex = (currentPage - 1) * PAGE_SIZE;
  const endIndex = Math.min(startIndex + PAGE_SIZE, totalItems);
  const pagedData = filteredData.slice(startIndex, endIndex);

  // 표시할 페이지 번호 범위 계산 (최대 5개)
  const pageNumbers = useMemo(() => {
    const maxVisible = 5;
    let start = Math.max(1, currentPage - Math.floor(maxVisible / 2));
    const end = Math.min(totalPages, start + maxVisible - 1);
    start = Math.max(1, end - maxVisible + 1);
    const pages: number[] = [];
    for (let i = start; i <= end; i++) pages.push(i);
    return pages;
  }, [currentPage, totalPages]);

  const toggleSection = useCallback((key: string) => {
    setCollapsedSections(prev => {
      const next = { ...prev, [key]: !prev[key] };
      // 접기/펼치기 후 All Requests 테이블 높이 재계산
      setTimeout(calcTableHeight, 50);
      return next;
    });
  }, [calcTableHeight]);

  const getFilterDisplayName = (filter: string) => {
    switch (filter) {
      case 'traceId': return t('trace_id');
      case 'workload': return t('workload');
      case 'method': return t('method');
      case 'status': return t('status');
      default: return t('filter');
    }
  };

  return (
    <div className="edge-detail-panel">
      <div className="edge-detail-panel-header">
        <div className="edge-detail-panel-title">
          <div className="connection-flow">
            <span className="source-node">{connectionInfo.sourceWorkload}</span>
            <span className="flow-arrow">
              <img src={nextIcon} alt="flow arrow" style={{ width: '16px', height: '16px' }}/>
            </span>
            <span className="target-node">{connectionInfo.targetWorkload}</span>
          </div>
          <div className="edge-detail-panel-subtitle">
            {connectionInfo.protocol}
          </div>
        </div>
        <button className="edge-detail-panel-close" onClick={onClose}>
          <img src={closeWindow} />
        </button>
      </div>

      <div className="edge-detail-panel-content">
        <div className={`trace-section-card ${collapsedSections['slow'] ? 'collapsed' : ''}`}>
          <h4 className="collapsible-header" onClick={() => toggleSection('slow')}>
            <span className={`collapse-arrow ${collapsedSections['slow'] ? '' : 'expanded'}`}>›</span>
            {t('termination_req')}
            <span className="section-count">{(traceData?.topSlowRequests || []).length}</span>
          </h4>
          {!collapsedSections['slow'] && (
            (traceData?.topSlowRequests || []).length > 0 ? (
              <KO11yTable
                items={{
                  headers: [
                    { name: t('time'), accessor: 'time', minWidth: 200, textAlign: 'left' },
                    { name: t('trace_id'), accessor: 'traceId', minWidth: 300, textAlign: 'left' },
                    { name: t('path_method'), accessor: 'pathMethod', minWidth: 300, textAlign: 'left' },
                    { name: t('status'), accessor: 'httpStatusCode', minWidth: 130, textAlign: 'center' },
                    { name: t('latency_ms'), accessor: 'latency', minWidth: 120, textAlign: 'left' },
                  ],
                  body: (traceData?.topSlowRequests || []).map(slow => ({
                    time: formatTimezoneAdjustedTimestamp(slow.timestamp),
                    traceId: <TraceIdCell traceId={slow.traceId} onClick={onTraceClick} />,
                    pathMethod: `${slow.method} ${slow.path}`,
                    httpStatusCode: { status: slow.status, isError: slow.isError, protocol: connectionInfo.protocol },
                    latency: formatLatency(slow.latency),
                    link: '',
                  }))
                }}
                tappable={false}
                footer={false}
                firstCellDivider={false}
                tableHeight="200px"
                bodyStyle={{
                  backgroundColor: 'transparent',
                  border: 'none'
                }}
              />
            ) : (
              <div className="empty-section-message">No slow requests</div>
            )
          )}
        </div>

        <div className={`trace-section-card ${collapsedSections['error'] ? 'collapsed' : ''}`}>
          <h4 className="collapsible-header" onClick={() => toggleSection('error')}>
            <span className={`collapse-arrow ${collapsedSections['error'] ? '' : 'expanded'}`}>›</span>
            {t('originating_req')}
            <span className="section-count">{(traceData?.recentErrors || []).length}</span>
          </h4>
          {!collapsedSections['error'] && (
            (traceData?.recentErrors || []).length > 0 ? (
              <KO11yTable
                items={{
                  headers: [
                    { name: t('time'), accessor: 'time', minWidth: 200, textAlign: 'left' },
                    { name: t('trace_id'), accessor: 'traceId', minWidth: 300, textAlign: 'left' },
                    { name: t('path_method'), accessor: 'pathMethod', minWidth: 300, textAlign: 'left' },
                    { name: t('status'), accessor: 'httpStatusCode', minWidth: 130, textAlign: 'center' },
                    { name: t('latency_ms'), accessor: 'latency', minWidth: 120, textAlign: 'left' },
                  ],
                  body: (traceData?.recentErrors || []).map(error => ({
                    time: formatTimezoneAdjustedTimestamp(error.timestamp),
                    traceId: <TraceIdCell traceId={error.traceId} onClick={onTraceClick} />,
                    pathMethod: `${error.method} ${error.path}`,
                    httpStatusCode: { status: error.status, isError: error.isError, protocol: connectionInfo.protocol },
                    latency: formatLatency(error.latency),
                    link: '',
                  }))
                }}
                tappable={false}
                footer={false}
                firstCellDivider={false}
                tableHeight="200px"
                bodyStyle={{
                  backgroundColor: 'transparent',
                  border: 'none'
                }}
              />
            ) : (
              <div className="empty-section-message">No error requests</div>
            )
          )}
        </div>

        <div className={`trace-section ${collapsedSections['all'] ? 'collapsed' : ''}`} ref={traceSectionRef}>
          <div className="section-header">
            <h4 className="collapsible-header" onClick={() => toggleSection('all')}>
              <span className={`collapse-arrow ${collapsedSections['all'] ? '' : 'expanded'}`}>›</span>
              {t('all_req')}
              <span className="section-count">{totalItems}</span>
            </h4>
            {!collapsedSections['all'] && (
              <div className="section-controls">
                <div className="filter-dropdown-container">
                  <button
                    className="filter-button-new"
                    onClick={() => setShowFilterDropdown(!showFilterDropdown)}
                  >
                    <img src={filterLightmode} alt={t('filter')} width={16} height={16} />
                    {getFilterDisplayName(selectedFilter)}
                  </button>
                  {showFilterDropdown && (
                    <div className="filter-dropdown">
                      <div className="filter-option" onClick={() => {setSelectedFilter('traceId'); setShowFilterDropdown(false);}}>
                        {t('trace_id')}
                      </div>
                      <div className="filter-option" onClick={() => {setSelectedFilter('workload'); setShowFilterDropdown(false);}}>
                        {t('workload')}
                      </div>
                      <div className="filter-option" onClick={() => {setSelectedFilter('method'); setShowFilterDropdown(false);}}>
                        {t('method')}
                      </div>
                      <div className="filter-option" onClick={() => {setSelectedFilter('status'); setShowFilterDropdown(false);}}>
                        {t('status')}
                      </div>
                      <div className="filter-option" onClick={() => {setSelectedFilter(''); setShowFilterDropdown(false);}}>
                        {t('clear_filter')}
                      </div>
                    </div>
                  )}
                </div>
                <div className="search-input-container">
                  <Input
                    type="text"
                    placeholder={t('search')}
                    value={searchText}
                    onChange={(e) => setSearchText(e.target.value)}
                    prefix={<img src={searchSvg} alt={t('search')} style={{ width: '14px', height: '14px' }} />}
                    style={{
                      width: '240px',
                      height: '36px',
                      borderRadius: '8px',
                    }}
                  />
                </div>
              </div>
            )}
          </div>
          {!collapsedSections['all'] && (
            <>
              {totalItems > 0 ? (
                <KO11yTable
                  items={{
                    headers: [
                      { name: t('time'), accessor: 'time', minWidth: 200, textAlign: 'left' },
                      { name: t('trace_id'), accessor: 'traceId', minWidth: 300, textAlign: 'left' },
                      { name: t('workload'), accessor: 'workload', minWidth: 120, textAlign: 'left' },
                      { name: t('method'), accessor: 'method', minWidth: 70, textAlign: 'center' },
                      { name: t('path'), accessor: 'path', minWidth: 250, maxWidth: 250, textAlign: 'left', showTooltip: true },
                      { name: t('status'), accessor: 'httpStatusCode', minWidth: 70, textAlign: 'center' },
                      { name: t('latency_ms'), accessor: 'latency', minWidth: 100, textAlign: 'left' },
                      { name: t('protocol'), accessor: 'protocol', minWidth: 80, textAlign: 'center' },
                    ],
                    body: pagedData.map(trace => ({
                      time: trace.time,
                      traceId: <TraceIdCell traceId={trace.traceId} onClick={onTraceClick} />,
                      workload: connectionInfo.sourceWorkload,
                      method: trace.method,
                      path: <PathCell path={trace.path} />,
                      httpStatusCode: { status: trace.status, isError: trace.isError, protocol: connectionInfo.protocol },
                      latency: formatLatency(trace.latency),
                      protocol: connectionInfo.protocol,
                      link: '',
                    }))
                  }}
                  tappable={false}
                  footer={false}
                  firstCellDivider={false}
                  tableHeight={allReqTableHeight}
                  bodyStyle={{
                    backgroundColor: 'transparent',
                    border: 'none'
                  }}
                />
              ) : (
                <div className="empty-section-message">
                  {searchText ? 'No matching requests' : 'No requests'}
                </div>
              )}
              {totalItems > 0 && (
                <div className="pagination-container">
                  <span className="pagination-info">
                    {startIndex + 1}-{endIndex} of {totalItems}
                  </span>
                  <div className="pagination-controls">
                    <button
                      className="pagination-btn"
                      disabled={currentPage === 1}
                      onClick={() => setCurrentPage(prev => prev - 1)}
                    >
                      ‹
                    </button>
                    {pageNumbers[0] > 1 && (
                      <>
                        <button className="pagination-btn" onClick={() => setCurrentPage(1)}>1</button>
                        {pageNumbers[0] > 2 && <span className="pagination-ellipsis">…</span>}
                      </>
                    )}
                    {pageNumbers.map(page => (
                      <button
                        key={page}
                        className={`pagination-btn ${page === currentPage ? 'active' : ''}`}
                        onClick={() => setCurrentPage(page)}
                      >
                        {page}
                      </button>
                    ))}
                    {pageNumbers[pageNumbers.length - 1] < totalPages && (
                      <>
                        {pageNumbers[pageNumbers.length - 1] < totalPages - 1 && <span className="pagination-ellipsis">…</span>}
                        <button className="pagination-btn" onClick={() => setCurrentPage(totalPages)}>{totalPages}</button>
                      </>
                    )}
                    <button
                      className="pagination-btn"
                      disabled={currentPage === totalPages}
                      onClick={() => setCurrentPage(prev => prev + 1)}
                    >
                      ›
                    </button>
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* 로딩 오버레이 */}
      {isLoading && (
        <NetworkMapLoader
          message="Loading connection details..."
          size="large"
        />
      )}
    </div>
  );
}, (prevProps, nextProps) => {
  // timeRange 변경 감지를 위한 커스텀 비교
  return (
    prevProps.edgeData.id === nextProps.edgeData.id &&
    prevProps.timeRange.start === nextProps.timeRange.start &&
    prevProps.timeRange.end === nextProps.timeRange.end
  );
});

NetworkEdgeDetailPanel.displayName = 'NetworkEdgeDetailPanel';

export default NetworkEdgeDetailPanel;