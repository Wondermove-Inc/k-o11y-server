export const slackTitleDefaultValue = [
	'{{- $severity := .CommonLabels.severity -}}',
	'{{- if eq $severity "critical" -}}:경광등: [CRITICAL] Alert Triggered :경광등:{{- else if eq $severity "error" -}}👾 [ERROR] Alert Triggered 👾{{- else if eq $severity "warning" -}}⚠️ [WARNING] Alert Triggered ⚠️{{- else -}}ℹ️ [INFO] Alert Triggered ℹ️{{- end }}',
].join('\n');

export const slackDescriptionDefaultValue = [
	'{{ range .Alerts -}}',
	'*Alert:* {{ .Labels.alertname }}{{ if .Labels.severity }} - {{ .Labels.severity }}{{ end }}',
	'*Status:* {{ .Status }}',
	'*Summary:* {{ .Annotations.summary }}',
	'*Description:* {{ .Annotations.description }}',
	'',
	'{{- if gt (len .Annotations.related_logs) 0 }}',
	'*Type:* Logs Alert',
	'*Links:*',
	'• Rule: <{{ .Labels.ruleSource }}|view rule>',
	'• Logs: <{{ .Annotations.related_logs }}|logs>',
	'{{- else if gt (len .Annotations.related_traces) 0 }}',
	'*Type:* Traces Alert',
	'*Links:*',
	'• Rule: <{{ .Labels.ruleSource }}|view rule>',
	'• Traces: <{{ .Annotations.related_traces }}|traces>',
	'{{- else }}',
	'*Type:* Metrics Alert',
	'*Links:*',
	'• Rule: <{{ .Labels.ruleSource }}|view rule>',
	'{{- end }}',
	'',
	'*Details:*',
	'{{ range .Labels.SortedPairs }} • *{{ .Name }}:* {{ .Value }}',
	'{{ end }}',
	'{{ end }}',
].join('\n');

export const editSlackDescriptionDefaultValue = slackDescriptionDefaultValue;

const dummySlackConfig = {
	api_url:
		'https://discord.com/api/webhooks/dummy_webhook_id/dummy_webhook_token/slack',
	channel: '#dummy_channel',
	send_resolved: true,
	text: slackDescriptionDefaultValue,
	title: slackTitleDefaultValue,
};

export const allAlertChannels = [
	{
		id: '3',
		created_at: '2023-08-09T04:45:19.239344617Z',
		updated_at: '2024-06-27T11:37:14.841184399Z',
		name: 'Dummy-Channel',
		type: 'slack',
		data: JSON.stringify({
			name: 'Dummy-Channel',
			slack_configs: [dummySlackConfig],
		}),
	},
];

export const editAlertChannelInitialValue = {
	...dummySlackConfig,
	type: 'slack',
	name: 'Dummy-Channel',
};

export const pagerDutyDescriptionDefaultVaule = `{{ if gt (len .Alerts.Firing) 0 -}} Alerts Firing: {{ range .Alerts.Firing }} - Message: {{ .Annotations.description }} Labels: {{ range .Labels.SortedPairs }} - {{ .Name }} = {{ .Value }} {{ end }} Annotations: {{ range .Annotations.SortedPairs }} - {{ .Name }} = {{ .Value }} {{ end }} Source: {{ .GeneratorURL }} {{ end }} {{- end }} {{ if gt (len .Alerts.Resolved) 0 -}} Alerts Resolved: {{ range .Alerts.Resolved }} - Message: {{ .Annotations.description }} Labels: {{ range .Labels.SortedPairs }} - {{ .Name }} = {{ .Value }} {{ end }} Annotations: {{ range .Annotations.SortedPairs }} - {{ .Name }} = {{ .Value }} {{ end }} Source: {{ .GeneratorURL }} {{ end }} {{- end }}`;

export const pagerDutyAdditionalDetailsDefaultValue = JSON.stringify({
	firing: `{{ template "pagerduty.default.instances" .Alerts.Firing }}`,
	resolved: `{{ template "pagerduty.default.instances" .Alerts.Resolved }}`,
	num_firing: '{{ .Alerts.Firing | len }}',
	num_resolved: '{{ .Alerts.Resolved | len }}',
});

export const opsGenieMessageDefaultValue = `{{ .CommonLabels.alertname }}`;

export const opsGenieDescriptionDefaultValue = `{{ if gt (len .Alerts.Firing) 0 -}} Alerts Firing: {{ range .Alerts.Firing }} - Message: {{ .Annotations.description }} Labels: {{ range .Labels.SortedPairs }} - {{ .Name }} = {{ .Value }} {{ end }} Annotations: {{ range .Annotations.SortedPairs }} - {{ .Name }} = {{ .Value }} {{ end }} Source: {{ .GeneratorURL }} {{ end }} {{- end }} {{ if gt (len .Alerts.Resolved) 0 -}} Alerts Resolved: {{ range .Alerts.Resolved }} - Message: {{ .Annotations.description }} Labels: {{ range .Labels.SortedPairs }} - {{ .Name }} = {{ .Value }} {{ end }} Annotations: {{ range .Annotations.SortedPairs }} - {{ .Name }} = {{ .Value }} {{ end }} Source: {{ .GeneratorURL }} {{ end }} {{- end }}`;

export const opsGeniePriorityDefaultValue =
	'{{ if eq (index .Alerts 0).Labels.severity "critical" }}P1{{ else if eq (index .Alerts 0).Labels.severity "warning" }}P2{{ else if eq (index .Alerts 0).Labels.severity "info" }}P3{{ else }}P4{{ end }}';

export const pagerDutySeverityTextDefaultValue =
	'{{ (index .Alerts 0).Labels.severity }}';
