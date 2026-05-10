export type JsonRecord = Record<string, unknown>;

export type AdminRole = "super_admin" | "admin";

export type HealthResponse = {
  status: string;
};

export type StatusResponse = {
  status: string;
};

export type AdminUser = {
  id: number;
  email: string;
  role: AdminRole;
  is_active: boolean;
};

export type LoginRequest = {
  email: string;
  password: string;
};

export type AuthResponse = {
  user: AdminUser;
};

export type CreateAdminUserRequest = {
  email: string;
  password: string;
};

export type Channel = {
  id: number;
  kick_channel_id: number | null;
  kick_chatroom_id: number | null;
  slug: string;
  display_name: string;
  profile_image_url: string | null;
  banner_image_url: string | null;
  is_enabled: boolean;
};

export type AddChannelRequest = {
  slug: string;
};

export type MessageEmote = {
  id: string;
  name: string;
  token: string;
  image_url: string;
};

export type MessageSender = {
  id: number;
  kick_user_id: number;
  username: string;
  slug: string;
  profile_image_url: string | null;
};

export type MessageChannel = {
  id: number;
  slug: string;
  display_name: string;
  profile_image_url: string | null;
  banner_image_url: string | null;
};

export type Message = {
  id: number;
  kick_message_id: string;
  chatroom_id: number;
  content: string;
  message_type: string;
  sender_username_snapshot: string;
  sender_slug_snapshot: string;
  sender_color_snapshot: string | null;
  sender_badges: JsonRecord[];
  emotes: MessageEmote[];
  reply_metadata: JsonRecord;
  thread_parent_id: string | null;
  message_created_at: string;
  ingested_at: string;
  sender: MessageSender;
  channel: MessageChannel;
};

export type MessageSearchParams = {
  sender?: string;
  channel?: string;
  q?: string;
  start?: string;
  end?: string;
  cursor?: string;
  limit?: number;
};

export type MessageSearchResponse = {
  items: Message[];
  next_cursor: string | null;
};

export type ApiErrorBody = {
  detail?: unknown;
};
