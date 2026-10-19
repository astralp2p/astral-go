package apphost

const DefaultEndpoint = "tcp:127.0.0.1:8625"
const AuthTokenEnv = "ASTRALD_APPHOST_TOKEN"

// EndpointEnv names the apphost endpoint the default router dials; unset, it
// dials DefaultEndpoint.
const EndpointEnv = "ASTRALD_APPHOST_ENDPOINT"
