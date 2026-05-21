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
  message_count: number;
  last_message_at: string | null;
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

export type IngestionHealth = {
  queue_depth: number;
  oldest_pending_age_seconds: number;
  write_queue_depth: number;
  write_queue_high_water_mark: number;
  write_drop_count: number;
  write_flush_count: number;
  last_flush_size: number;
  last_flush_millis: number;
  clickhouse_insert_failures: number;
  queue_enqueue_failures: number;
  breaker_state: string;
  breaker_current_delay_ms: number;
};

export type OperationsSummary = {
  counts: OperationsCounts;
  raw_event_status_counts: RawEventStatusCounts;
  storage: OperationsStorage;
  timestamps: OperationsTimestamps;
  listener: ListenerHeartbeat;
  ingestion: IngestionHealth;
};

export type FailedRawEvent = {
  raw_event_id: string;
  channel_slug: string;
  error_message: string;
  attempts: number;
  received_at: string;
  failed_at: string;
};

export type FailedRawEventsResponse = {
  events: FailedRawEvent[];
  total: number;
};

export type FailedEventsActionResponse = {
  affected: number;
  message: string;
};

export type RetentionDays = 30 | 90 | null;

export type RetentionSettings = {
  message_retention_days: RetentionDays;
  raw_event_retention_days: RetentionDays;
  updated_at: string | null;
};

export type UpdateRetentionSettingsRequest = {
  message_retention_days: RetentionDays;
  raw_event_retention_days: RetentionDays;
};

export type DataManagementCounts = {
  channels: number;
  senders: number;
  messages: number;
  raw_events: number;
};

export type DataManagementTable = {
  table_name: string;
  total_bytes: number;
  row_count: number;
};

export type DataManagementSummary = {
  counts: DataManagementCounts;
  database_bytes: number;
  tables: DataManagementTable[];
  retention_settings: RetentionSettings;
};

export type DataCleanupTarget = "old_messages" | "old_raw_events" | "channel" | "sender";

export type DataCleanupRequest = {
  target: DataCleanupTarget;
  channel_slug?: string | null;
  sender?: string | null;
};

export type DataCleanupConfirmRequest = DataCleanupRequest & {
  confirmation_text: string;
};

export type DataCleanupCounts = {
  messages: number;
  raw_events: number;
  total: number;
};

export type DataCleanupPreview = {
  target: DataCleanupTarget;
  affected: DataCleanupCounts;
  confirmation_text: string;
  can_execute: boolean;
  cutoff_at: string | null;
  channel_slug: string | null;
  sender: string | null;
  retention_days: number | null;
  reason: string | null;
};

export type DataCleanupResult = {
  target: DataCleanupTarget;
  deleted: DataCleanupCounts;
  confirmation_text: string;
  cutoff_at: string | null;
  channel_slug: string | null;
  sender: string | null;
  retention_days: number | null;
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
