import 'dart:async';
import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:web_socket_channel/web_socket_channel.dart';
import '../models/message.dart';

class MessagingService {
  final String httpBaseUrl;
  final String wsBaseUrl;
  final String token;

  WebSocketChannel? _channel;
  final StreamController<Message> _messageController = StreamController<Message>.broadcast();

  MessagingService({
    required this.httpBaseUrl,
    required this.wsBaseUrl,
    required this.token,
  });

  Stream<Message> get messageStream => _messageController.stream;

  void connect() {
    if (token.isEmpty) {
      print('WS connect skipped: missing token');
      return;
    }
    final uri = Uri.parse('$wsBaseUrl/ws?token=$token');
    _channel = WebSocketChannel.connect(uri);

    _channel!.stream.listen((data) {
      try {
        final decoded = jsonDecode(data);
        final msg = Message.fromJson(decoded);
        _messageController.add(msg);
      } catch (e) {
        print('Error parsing WS message: $e');
      }
    }, onDone: () {
      print('WS connection closed. Reconnecting...');
      Future.delayed(Duration(seconds: 3), () => connect());
    }, onError: (e) {
      print('WS error: $e');
    });
  }

  Future<List<Map<String, dynamic>>> getMyChannels() async {
    try {
      final response = await http.get(
        Uri.parse('$httpBaseUrl/channels'),
        headers: {
          if (token.isNotEmpty) 'Authorization': 'Bearer $token',
        },
      );
      if (response.statusCode == 200) {
        final data = jsonDecode(response.body) as Map<String, dynamic>;
        return List<Map<String, dynamic>>.from(data['channels'] ?? const []);
      }
      return [];
    } catch (e) {
      print('Channels error: $e');
      return [];
    }
  }

  Future<String?> createDMChannel(String targetUserId) async {
    try {
      final response = await http.post(
        Uri.parse('$httpBaseUrl/dm'),
        headers: {
          'Content-Type': 'application/json',
          if (token.isNotEmpty) 'Authorization': 'Bearer $token',
        },
        body: jsonEncode({'target_user_id': targetUserId}),
      );
      if (response.statusCode == 200) {
        final data = jsonDecode(response.body) as Map<String, dynamic>;
        return data['channel_id'] as String?;
      }
      return null;
    } catch (e) {
      print('Create DM error: $e');
      return null;
    }
  }

  Future<List<Map<String, dynamic>>> getChannelMembers(String channelId) async {
    try {
      final response = await http.get(
        Uri.parse('$httpBaseUrl/channel-members?channel_id=$channelId'),
        headers: {
          if (token.isNotEmpty) 'Authorization': 'Bearer $token',
        },
      );
      if (response.statusCode == 200) {
        final data = jsonDecode(response.body) as Map<String, dynamic>;
        return List<Map<String, dynamic>>.from(data['members'] ?? const []);
      }
      return [];
    } catch (e) {
      print('Channel members error: $e');
      return [];
    }
  }

  Future<List<Message>> getHistory(String channelId) async {
    try {
      final response = await http.get(
        Uri.parse('$httpBaseUrl/history?channel_id=$channelId'),
        headers: {
          if (token.isNotEmpty) 'Authorization': 'Bearer $token',
        },
      );

      if (response.statusCode == 200) {
        final List data = jsonDecode(response.body);
        return data.map((m) => Message.fromJson(m)).toList();
      }
      return [];
    } catch (e) {
      print('History error: $e');
      return [];
    }
  }

  Future<bool> sendMessage(String channelId, String content, MessageType type) async {
    try {
      final payload = {
        'channel_id': channelId,
        'content': base64Encode(utf8.encode(content)),
        'type': type.index + 1, // 1=Text, etc.
        'nonce': '',
        'signature': '',
      };

      // Try sending via WebSocket if connected, else fallback to HTTP
      if (_channel != null) {
        _channel!.sink.add(jsonEncode(payload));
        return true;
      }

      final response = await http.post(
        Uri.parse('$httpBaseUrl/send'),
        body: jsonEncode(payload),
        headers: {
          'Content-Type': 'application/json',
          if (token.isNotEmpty) 'Authorization': 'Bearer $token',
        },
      );

      return response.statusCode == 200;
    } catch (e) {
      print('Send error: $e');
      return false;
    }
  }

  void dispose() {
    _channel?.sink.close();
    _messageController.close();
  }
}
