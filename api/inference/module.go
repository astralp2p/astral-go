// Package inference holds the wire types of the inference protocol: what a
// provider offers, one completion request, and the deltas of its answer.
//
// The types are deliberately not an OpenAI REST mirror. An astral offering is
// already caller-specific, a channel already streams, and an object ID already
// addresses media network-wide, so the wire carries only what those primitives
// do not: the model inventory, a request, and the deltas of a stream.
package inference

// ServiceName is the services name a provider advertises.
const ServiceName = "inference"

// Op names. An offering's services.operations_list carries them, and a
// consumer addresses them to the provider identity.
const (
	OpModels   = "inference.models"
	OpComplete = "inference.complete"
)

// Message roles.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// Content part kinds.
const (
	PartText  = "text"
	PartImage = "image"
	PartAudio = "audio"
	PartVideo = "video"
)

// Source kinds of a media part: bytes carried inline, an object on the astral
// network, or a URL the provider fetches itself.
const (
	SourceInline = "inline"
	SourceObject = "object"
	SourceURL    = "url"
)

// Effort level names with agreed meaning. A provider may offer others.
const (
	// EffortOff means the model does not think.
	EffortOff    = "off"
	EffortLow    = "low"
	EffortMedium = "medium"
	EffortHigh   = "high"
)

// Tool choices. An empty ToolChoice means ToolChoiceAuto; any other value
// names the one tool the model must call.
const (
	ToolChoiceAuto     = "auto"
	ToolChoiceNone     = "none"
	ToolChoiceRequired = "required"
)

// Finish reasons with agreed meaning. A provider passes its server's reason
// through, so a consumer meets others too.
const (
	FinishStop          = "stop"
	FinishLength        = "length"
	FinishToolCalls     = "tool_calls"
	FinishContentFilter = "content_filter"
)

// Error kinds.
const (
	ErrorRefused         = "refused"
	ErrorBadRequest      = "bad_request"
	ErrorContextOverflow = "context_overflow"
	ErrorRateLimited     = "rate_limited"
	ErrorUnavailable     = "unavailable"
	// ErrorFarSide means the server behind the provider failed.
	ErrorFarSide = "far_side"
)
