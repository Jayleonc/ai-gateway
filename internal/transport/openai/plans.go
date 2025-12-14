package openai

type ErrorPlan struct {
	HTTPStatus int
	Body       any
}

type StreamEndAction string

const (
	StreamEndActionNone           StreamEndAction = "none"
	StreamEndActionWriteJSONError StreamEndAction = "write_json_error"
	StreamEndActionCloseStream    StreamEndAction = "close_stream"
)

type StreamEndPlan struct {
	Action     StreamEndAction
	Error      *ErrorPlan
	HTTPStatus int
}
