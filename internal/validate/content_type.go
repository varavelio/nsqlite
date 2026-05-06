package validate

type (
	contentType     string
	contenttestType string
)

const (
	ContentTypePlainText contentType = "text/plain"
	ContentTypeJSON      contentType = "application/json"
	ContentTypeXML       contentType = "application/xml"
	ContentTypeHTML      contentType = "text/html"
	ContentTypeForm      contentType = "application/x-www-form-urlencoded"
	ContentTypeMultipart contentType = "multipart/form-data"
)

const ContentTypetestMultipart contenttestType = "multipart/form-data"

// ContentType returns true if the given content type is in the list
// of allowed content types.
func ContentType(
	contentTypeTarget string,
	allowedContentTypes ...contentType,
) bool {
	for _, allowedContentType := range allowedContentTypes {
		if contentTypeTarget == string(allowedContentType) {
			return true
		}
	}

	return false
}
