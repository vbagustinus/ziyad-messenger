import 'package:flutter/material.dart';
import '../services/messaging_service.dart';
import '../services/directory_service.dart';
import '../models/message.dart';
import '../models/user.dart';

class ChatProvider extends ChangeNotifier {
  final MessagingService messagingService;
  final DirectoryService directoryService;

  List<User> users = [];
  List<Map<String, dynamic>> channels = [];
  Map<String, List<Message>> messageHistory = {};
  String? currentChannelId;
  bool isLoading = false;

  ChatProvider({required this.messagingService, required this.directoryService}) {
    // Listen to real-time messages
    messagingService.messageStream.listen((msg) {
      if (!messageHistory.containsKey(msg.channelId)) {
        messageHistory[msg.channelId] = [];
      }
      messageHistory[msg.channelId]!.add(msg);
      notifyListeners();
    });
  }

  Future<void> loadDirectory() async {
    isLoading = true;
    notifyListeners();
    
    users = await directoryService.getUsers();
    channels = await directoryService.getChannels();
    
    isLoading = false;
    notifyListeners();
  }

  Future<void> selectChannel(String channelId) async {
    currentChannelId = channelId;
    if (!messageHistory.containsKey(channelId)) {
      final history = await messagingService.getHistory(channelId);
      messageHistory[channelId] = history;
    }
    notifyListeners();
  }

  Future<void> sendText(String content) async {
    if (currentChannelId == null) return;
    await messagingService.sendMessage(currentChannelId!, content, MessageType.text);
  }

  // Helper to get formatted name for a channel or user
  String getChannelName(String id) {
    final chan = channels.where((c) => c['id'] == id).firstOrNull;
    if (chan != null) return '# ${chan['name']}';
    
    final user = users.where((u) => u.id == id).firstOrNull;
    if (user != null) return '@ ${user.username}';
    
    return id;
  }
}
