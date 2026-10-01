export type DirectoryKind = "users" | "channels";

export type DirectoryIdentity = {
  id: number;
  name: string;
  slug: string;
  profile_image_url: string | null;
};

export type DirectoryPage = {
  items: DirectoryIdentity[];
  next_cursor: string | null;
};

export type DirectoryQuery = {
  prefix: string;
  limit?: number;
  after?: string;
};
