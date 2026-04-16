import { EmailChannel, OpsgenieChannel, PagerChannel } from './config';

export const PagerInitialConfig: Partial<PagerChannel> = {
	description: `[{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}] {{ .CommonLabels.alertname }} for {{ .CommonLabels.job }}
	{{- if gt (len .CommonLabels) (len .GroupLabels) -}}
	  {{" "}}(
	  {{- with .CommonLabels.Remove .GroupLabels.Names }}
		{{- range $index, $label := .SortedPairs -}}
		  {{ if $index }}, {{ end }}
		  {{- $label.Name }}="{{ $label.Value -}}"
		{{- end }}
	  {{- end -}}
	  )
	{{- end }}`,
	severity: '{{ (index .Alerts 0).Labels.severity }}',
	client: 'SigNoz Alert Manager',
	client_url: 'https://enter-signoz-host-n-port-here/alerts',
	details: JSON.stringify({
		firing: `{{ template "pagerduty.default.instances" .Alerts.Firing }}`,
		resolved: `{{ template "pagerduty.default.instances" .Alerts.Resolved }}`,
		num_firing: '{{ .Alerts.Firing | len }}',
		num_resolved: '{{ .Alerts.Resolved | len }}',
	}),
};

export const OpsgenieInitialConfig: Partial<OpsgenieChannel> = {
	message: '{{ .CommonLabels.alertname }}',
	description: `{{ if gt (len .Alerts.Firing) 0 -}}
	Alerts Firing:
	{{ range .Alerts.Firing }}
	 - Message: {{ .Annotations.description }}
	Labels:
	{{ range .Labels.SortedPairs }}   - {{ .Name }} = {{ .Value }}
	{{ end }}   Annotations:
	{{ range .Annotations.SortedPairs }}   - {{ .Name }} = {{ .Value }}
	{{ end }}   Source: {{ .GeneratorURL }}
	{{ end }}
	{{- end }}
	{{ if gt (len .Alerts.Resolved) 0 -}}
	Alerts Resolved:
	{{ range .Alerts.Resolved }}
	 - Message: {{ .Annotations.description }}
	Labels:
	{{ range .Labels.SortedPairs }}   - {{ .Name }} = {{ .Value }}
	{{ end }}   Annotations:
	{{ range .Annotations.SortedPairs }}   - {{ .Name }} = {{ .Value }}
	{{ end }}   Source: {{ .GeneratorURL }}
	{{ end }}
	{{- end }}`,
	priority:
		'{{ if eq (index .Alerts 0).Labels.severity "critical" }}P1{{ else if eq (index .Alerts 0).Labels.severity "warning" }}P2{{ else if eq (index .Alerts 0).Labels.severity "info" }}P3{{ else }}P4{{ end }}',
};

export const EmailInitialConfig: Partial<EmailChannel> = {
	send_resolved: true,
	html: `<!--
	Credits: https://github.com/mailgun/transactional-email-templates
	-->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="initial-scale=1.0,user-scalable=yes">
  <title>{{ template "__subject" . }}</title>
</head>
<body style="background-color: #E7EEF6; margin: 0; padding: 0;">
  <table width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color: #E7EEF6;">
    <tr>
      <td style="background-color: #E7EEF6;">
        <table style="width: 100%; border-spacing: 0; border-collapse: collapse; margin: 40px 0;">
          <tbody>
            <tr>
              <th colspan="5" style="text-align: center; padding: 20px 0; background-color: #E7EEF6;">
                <img src="" alt="Logo"/>
              </th>
            </tr>
            <tr>
              <td style="width: 20%; background-color: #E7EEF6;" bgcolor="#E7EEF6">&nbsp;</td>
              <td style="width: 60%; background-color: white; border-radius: 8px; padding: 30px; box-sizing: border-box;" bgcolor="white" colspan="3">
                <div style="padding: 0 0 32px 0; text-align: left; font-family: Pretendard, sans-serif;">
                      {{ range .Alerts -}}
                      {{ $severity := .Labels.severity -}}
                      {{- if eq $severity "critical" -}}
                      <p style="color: #1C1C1C; margin: 0 0 8px 0; font-family: Pretendard, sans-serif; font-size: 24px; font-weight: 700; line-height: 36px;">[Critical] K-O11y Alert</p>
                      <p style="color: #1C1C1C; margin: 0 0 16px 0; font-family: Pretendard, sans-serif; font-size: 14px; font-weight: 400; line-height: 20px;">
                        <strong>A critical alert is active.</strong><br>
                        Check the details below and respond as needed.
                      </p>
                      {{- else if eq $severity "error" -}}
                      <p style="color: #1C1C1C; margin: 0 0 8px 0; font-family: Pretendard, sans-serif; font-size: 24px; font-weight: 700; line-height: 36px;">[Error] K-O11y Alert</p>
                      <p style="color: #1C1C1C; margin: 0 0 16px 0; font-family: Pretendard, sans-serif; font-size: 14px; font-weight: 400; line-height: 20px;">
                        <strong>An error alert is active.</strong><br>
                        Quick response needed to address the issue.
                      </p>
                      {{- else if eq $severity "warning" -}}
                      <p style="color: #1C1C1C; margin: 0 0 8px 0; font-family: Pretendard, sans-serif; font-size: 24px; font-weight: 700; line-height: 36px;">[Warning] K-O11y Alert</p>
                      <p style="color: #1C1C1C; margin: 0 0 16px 0; font-family: Pretendard, sans-serif; font-size: 14px; font-weight: 400; line-height: 20px;">
                        <strong>A warning alert is active.</strong><br>
                        Please review and take appropriate action.
                      </p>
                      {{- else -}}
                      <p style="color: #1C1C1C; margin: 0 0 8px 0; font-family: Pretendard, sans-serif; font-size: 24px; font-weight: 700; line-height: 36px;">[Info] K-O11y</p>
                      <p style="color: #1C1C1C; margin: 0 0 16px 0; font-family: Pretendard, sans-serif; font-size: 14px; font-weight: 400; line-height: 20px;">
                        <strong>Informational update.</strong><br>
                        This is an informational notification for your reference.
                      </p>
                      {{- end }}
                      {{ end }}
                      
                      {{ if gt (len .Alerts.Firing) 0 }}
                      {{ range $index, $alert := .Alerts.Firing }}
                        <table width="100%" cellpadding="0" cellspacing="0" style="border: 1px solid #E7EEF6; border-radius: 8px; background-color: rgba(221, 221, 221, 0.10); margin-bottom: 16px; overflow: hidden;">
                          <tr>
                            <td style="color: #1C1C1C; font-weight: 600; padding: 12px 16px; width: 120px; font-family: Pretendard, sans-serif; font-size: 14px; vertical-align: top; border-bottom: 1px solid #E7EEF6;">Alert Name</td>
                            <td style="padding: 12px 16px; font-family: Pretendard, sans-serif; font-size: 14px; color: #374151; border-bottom: 1px solid #E7EEF6;">{{ index .Labels "alertname" }}</td>
                          </tr>
                          <tr>
                            <td style="color: #1C1C1C; font-weight: 600; padding: 12px 16px; width: 120px; font-family: Pretendard, sans-serif; font-size: 14px; vertical-align: top;">Severity</td>
                            <td style="padding: 12px 16px; font-family: Pretendard, sans-serif; font-size: 14px;">
                              {{ $severity := index .Labels "severity" }}
                              {{ if eq $severity "critical" }}
                              <span style="color: #EB4136; font-weight: 600;">{{ $severity }}</span>
                              {{ else if eq $severity "error" }}
                              <span style="color: #EB4136; font-weight: 600;">{{ $severity }}</span>
                              {{ else if eq $severity "warning" }}
                              <span style="color: #FFA800; font-weight: 600;">{{ $severity }}</span>
                              {{ else if eq $severity "info" }}
                              <span style="color: #125AED; font-weight: 600;">{{ $severity }}</span>
                              {{ end }}
                            </td>
                          </tr>
                        </table>

                        {{ if gt (len .Annotations) 0 }}
                        <h3 style="color: #1C1C1C; border-bottom: 1px solid #E7EEF6; font-family: Pretendard, sans-serif; font-size: 16px; font-weight: 600; padding: 24px 0 8px 0; margin: 0;">
                          Alert Detail
                        </h3>
                        <table width="100%" cellpadding="0" cellspacing="0" style="border-bottom: 1px solid #E7EEF6; overflow: hidden;">
                          {{ $isFirst := true }}
                          {{ range .Annotations.SortedPairs }}
                          {{ if ne .Name "runbook_url" }}
                          <tr{{ if not $isFirst }} style="border-top: 1px solid #E7EEF6;"{{ end }}>
                            <td style="color: #1C1C1C; font-weight: 600; padding: 12px 16px; width: 120px; font-family: Pretendard, sans-serif; font-size: 14px; vertical-align: top; text-transform: capitalize;">{{ .Name }}</td>
                            <td style="color: #374151; font-weight: 400; padding: 12px 16px; font-family: Pretendard, sans-serif; font-size: 14px; line-height: 1.5;">{{ .Value }}</td>
                          </tr>
                          {{ $isFirst = false }}
                          {{ end }}
                          {{ end }}
                        </table>
                        {{ end }}

                      {{ end }}
                      {{ end }}

                </div>
              </td>
              <td style="width: 20%; background-color: #E7EEF6;" bgcolor="#E7EEF6">&nbsp;</td>
            </tr>
            <tr>
              <td colspan="5" style="background-color: #E7EEF6; padding-top: 20px;" bgcolor="#E7EEF6"></td>
            </tr>
            <tr>
              <td bgcolor="#E7EEF6"></td>
              <td colspan="2" style="background-color: #E7EEF6; text-align: left; padding-bottom: 32px; color: rgba(37, 46, 64, 0.30); font-family: Pretendard, sans-serif; font-size: 12px; line-height: 16px;" bgcolor="#E7EEF6">
                © 2026 K-O11y, All rights reserved.<br>
                4F, 7-21, Gangnam-daero 27-gil, Seocho-gu, Seoul, Republic of Korea
              </td>
              <td colspan="1" style="background-color: #E7EEF6; text-align: right; padding-bottom: 32px;" bgcolor="#E7EEF6">
                <img src="" alt="Home Icon" style="width: 32px; height: 32px; vertical-align: middle;"/>
              </td>
              <td bgcolor="#E7EEF6"></td>
            </tr>
          </tbody>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`,
};
