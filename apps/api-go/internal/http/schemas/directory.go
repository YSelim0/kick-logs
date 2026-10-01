package schemas

type DirectoryIdentityResponse struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Slug            string  `json:"slug"`
	ProfileImageURL *string `json:"profile_image_url"`
}

type DirectoryResponse struct {
	Items      []DirectoryIdentityResponse `json:"items"`
	NextCursor *string                     `json:"next_cursor"`
}
