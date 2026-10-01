package domain

type DirectoryKind string

const (
	DirectoryUsers    DirectoryKind = "users"
	DirectoryChannels DirectoryKind = "channels"
)

type DirectoryIdentity struct {
	ID              int64
	Name            string
	Slug            string
	ProfileImageURL string
	SortSlug        string
}

type DirectoryPosition struct {
	Slug string
	ID   int64
}

type DirectoryQuery struct {
	Prefix string
	Limit  int
	After  DirectoryPosition
}

type DirectoryPage struct {
	Items      []DirectoryIdentity
	NextCursor string
}
