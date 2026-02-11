import 'dart:convert';

enum MessageType { unknown, text, image, file, system, voice }

class Message {
  final String id;
  final String channelId;
  final String senderId;
  final DateTime timestamp;
  final MessageType type;
  final String content; 
  final bool isMe;

  Message({
    required this.id,
    required this.channelId,
    required this.senderId,
    required this.timestamp,
    required this.type,
    required this.content,
    this.isMe = false,
  });

  factory Message.fromJson(Map<String, dynamic> json, {String? currentUserId}) {
    String rawContent = json['content'] ?? '';
    String decoded = rawContent;
    try {
      // Backend (messaging/main.go) encodes everything in base64
      decoded = utf8.decode(base64Decode(rawContent));
    } catch (_) { }

    return Message(
      id: json['id'] ?? '',
      channelId: json['channel_id'] ?? '',
      senderId: json['sender_id'] ?? '',
      timestamp: DateTime.fromMillisecondsSinceEpoch(json['timestamp'] ?? 0),
      type: _parseType(json['type']),
      content: decoded,
      isMe: (json['sender_id'] == currentUserId),
    );
  }

  static MessageType _parseType(dynamic type) {
    if (type is int && type >= 0 && type < MessageType.values.length) {
      return MessageType.values[type];
    }
    return MessageType.text;
  }
}
