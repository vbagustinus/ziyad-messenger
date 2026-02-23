/// API base URLs for the running backend services.
///
/// Defaults use localhost (works for macOS/Windows/Linux desktop and iOS simulator).
/// - Android emulator: run with
///   --dart-define=AUTH_BASE_URL=http://10.0.2.2:8086
///   --dart-define=MESSAGING_BASE_URL=http://10.0.2.2:8081
/// - Physical device: use your machine's LAN IP, e.g.
///   --dart-define=AUTH_BASE_URL=http://192.168.1.100:8086
class AppConfig {
  /// The central IP or Domain where the backend is running.
  /// Change this ONE line to point to a new server!
  static const String host = String.fromEnvironment(
    'BACKEND_HOST',
    defaultValue: '192.168.200.252',
  );

  static String get authBaseUrl => 'http://$host:8086';
  static String get messagingBaseUrl => 'http://$host:8081';
  static String get wsMessagingBaseUrl => 'ws://$host:8081';
  static String get presenceBaseUrl => 'http://$host:8083';
  static String get fileTransferBaseUrl => 'http://$host:8082';
  static String get adminApiBaseUrl => 'http://$host:8090';
}
