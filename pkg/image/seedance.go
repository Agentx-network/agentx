package image

import "context"

const (
	// Seedance/Seedream run on Volcengine Ark, which exposes an
	// OpenAI-compatible /images/generations endpoint. The base URL and exact
	// model id vary by region/account, so both are expected to be set in
	// Config → Image; these are sensible defaults.
	seedanceDefaultBase  = "https://ark.cn-beijing.volces.com/api/v3"
	seedanceDefaultModel = "seedance-1.0"
)

// generateSeedance generates an image via Volcengine Ark's OpenAI-compatible
// image API. The user supplies the Ark API key (and usually a base URL + the
// concrete model id) in Config → Image.
func generateSeedance(ctx context.Context, opts GenerateOptions) ([]byte, string, error) {
	return openAICompatImage(ctx, opts, seedanceDefaultBase, seedanceDefaultModel)
}
