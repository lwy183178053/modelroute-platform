export const ccSwitchUsageScript = `({
  request: {
    url: "{{baseUrl}}/v1/usage",
    method: "GET",
    headers: { "Authorization": "Bearer {{apiKey}}" }
  },
  extractor: function(response) {
    const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
    const unit = response?.unit ?? response?.quota?.unit ?? "CNY";
    return {
      isValid: response?.is_active ?? response?.isValid ?? false,
      remaining,
      unit
    };
  }
})`;

export function buildCCSwitchImportURL({ baseUrl, apiKey, model, provider, keyName }) {
  const origin = new URL(baseUrl).origin;
  const query = new URLSearchParams([
    ['resource', 'provider'],
    ['app', 'codex'],
    ['model', model],
    ['name', `${keyName} · ${provider}`],
    ['homepage', origin],
    ['endpoint', origin],
    ['apiKey', apiKey],
    ['configFormat', 'json'],
    ['usageEnabled', 'true'],
    ['usageScript', btoa(ccSwitchUsageScript)],
    ['usageAutoInterval', '30'],
  ]);
  return `ccswitch://v1/import?${query.toString()}`;
}
