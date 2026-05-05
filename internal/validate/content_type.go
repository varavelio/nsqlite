package validate

import "github.com/orsinium-labs/enum"

type (
	contentType     enum.Member[string]
	contenttestType enum.Member[string]
)

var (
	ContentTypePlainText = contentType{Value: "text/plain"}
	ContentTypeJSON      = contentType{Value: "application/json"}
	ContentTypeXML       = contentType{Value: "application/xml"}
	ContentTypeHTML      = contentType{Value: "text/html"}
	ContentTypeForm      = contentType{Value: "application/x-www-form-urlencoded"}
	ContentTypeMultipart = contentType{Value: "multipart/form-data"}

	ContentTypetestMultipart = contenttestType{Value: "multipart/form-data"}
)

// ContentType returns true if the given content type is in the list
// of allowed content types.
func ContentType(
	contentTypeTarget string,
	allowedContentTypes ...contentType,
) bool {
	for _, allowedContentType := range allowedContentTypes {
		if contentTypeTarget == allowedContentType.Value {
			return true
		}
	}

	return false
}
