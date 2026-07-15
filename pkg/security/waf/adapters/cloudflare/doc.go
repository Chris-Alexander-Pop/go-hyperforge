/*
Package cloudflare implements waf.Manager against the Cloudflare Firewall Rules API.

This is a thin control-plane adapter: BlockIP / AllowIP / GetRules map to
create / delete / list IP access rules via the Cloudflare HTTP API. Production
use requires a valid API token with Zone Firewall permissions. Tests inject
BaseURL + HTTPClient.
*/
package cloudflare
