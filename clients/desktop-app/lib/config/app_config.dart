/// API base URLs for the running backend services.
///
/// Defaults use localhost (works for macOS/Windows/Linux desktop and iOS simulator).
/// - Android emulator: run with
///   --dart-define=AUTH_BASE_URL=http://10.0.2.2:8086
///   --dart-define=MESSAGING_BASE_URL=http://10.0.2.2:8081
/// - Physical device: use your machine's LAN IP, e.g.
///   --dart-define=AUTH_BASE_URL=http://192.168.1.100:8086
class AppConfig {
  static const String authBaseUrl = String.fromEnvironment(
    'AUTH_BASE_URL',
    defaultValue: 'http://localhost:8086',
  );

  static const String messagingBaseUrl = String.fromEnvironment(
    'MESSAGING_BASE_URL',
    defaultValue: 'http://localhost:8081',
  );

  static const String presenceBaseUrl = String.fromEnvironment(
    'PRESENCE_BASE_URL',
    defaultValue: 'http://localhost:8083',
  );

  static const String fileTransferBaseUrl = String.fromEnvironment(
    'FILE_TRANSFER_BASE_URL',
    defaultValue: 'http://localhost:8082',
  );

  static const String adminApiBaseUrl = String.fromEnvironment(
    'ADMIN_API_BASE_URL',
    defaultValue: 'http://localhost:8090',
  );
}
