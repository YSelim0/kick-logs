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

export type OperationsCounts = {
  channels: number;
  enabled_channels: number;
  senders: number;
  messages: number;
  raw_events: number;
};

export type RawEventStatusCounts = Record<string, number>;

export type OperationsStorageTable = {
  table_name: string;
  total_bytes: number;
};

export type OperationsStorage = {
  database_bytes: number;
  tables: OperationsStorageTable[];
};

export type OperationsTimestamps = {
  latest_message_at: string | null;
  latest_raw_event_received_at: string | null;
  latest_raw_event_processed_at: string | null;
  oldest_pending_raw_event_received_at: string | null;
};

export type ListenerHeartbeat = {
  service_name: string;
  last_seen_at: string | null;
  is_fresh: boolean;
  stale_after_seconds: number;
  seconds_since_last_seen: number | null;
};

export type OperationsSummary = {
  counts: OperationsCounts;
  raw_event_status_counts: RawEventStatusCounts;
  storage: OperationsStorage;
  timestamps: OperationsTimestamps;
  listener: ListenerHeartbeat;
};

export type AnalyticsOverview = {
  total_messages: number;
  total_senders: number;
  total_channels: number;
  total_emote_usages: number;
  first_message_at: string | null;
  latest_message_at: string | null;
};

export type AnalyticsBucket = "hour" | "day";

export type MessageVolumePoint = {
  bucket_start: string;
  message_count: number;
};

export type MessageVolumeResponse = {
  items: MessageVolumePoint[];
};

export type TopSenderAnalytics = {
  sender_id: number;
  kick_user_id: number;
  username: string;
  slug: string;
  profile_image_url: string | null;
  message_count: number;
  first_message_at: string;
  latest_message_at: string;
};

export type TopSendersResponse = {
  items: TopSenderAnalytics[];
};

export type TopChannelAnalytics = {
  channel_id: number;
  slug: string;
  display_name: string;
  profile_image_url: string | null;
  banner_image_url: string | null;
  message_count: number;
  first_message_at: string;
  latest_message_at: string;
};

export type TopChannelsResponse = {
  items: TopChannelAnalytics[];
};

export type TopEmoteAnalytics = {
  id: string;
  name: string;
  token: string;
  image_url: string;
  usage_count: number;
  message_count: number;
};

export type TopEmotesResponse = {
  items: TopEmoteAnalytics[];
};

export type UserProfileSender = {
  id: number;
  kick_user_id: number;
  username: string;
  slug: string;
  profile_image_url: string | null;
};

export type UserProfile = {
  sender: UserProfileSender;
  overview: AnalyticsOverview;
  message_volume: MessageVolumePoint[];
  top_channels: TopChannelAnalytics[];
  top_emotes: TopEmoteAnalytics[];
  latest_messages: Message[];
};

export type ChannelProfileChannel = {
  id: number;
  kick_channel_id: number | null;
  kick_chatroom_id: number | null;
  slug: string;
  display_name: string;
  profile_image_url: string | null;
  banner_image_url: string | null;
  is_enabled: boolean;
};

export type ChannelProfile = {
  channel: ChannelProfileChannel;
  overview: AnalyticsOverview;
  message_volume: MessageVolumePoint[];
  top_senders: TopSenderAnalytics[];
  top_emotes: TopEmoteAnalytics[];
  latest_messages: Message[];
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
  reply_only?: boolean;
  emote_only?: boolean;
  cursor?: string;
  limit?: number;
};

export type MessageSearchResponse = {
  items: Message[];
  next_cursor: string | null;
};

export type MessageExportFormat = "csv" | "json";

export type MessageExportResponse = {
  items: Message[];
  count: number;
  max_rows: number;
  truncated: boolean;
};

export type ApiErrorBody = {
  detail?: unknown;
};
