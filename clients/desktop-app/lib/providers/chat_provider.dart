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
        
        // If it's a new channel/user we don't know about yet, refresh directory
        final knownChannel = channels.any((c) => c['id'] == msg.channelId);
        final knownUser = users.any((u) => u.id == msg.channelId);
        
        if (!knownChannel && !knownUser) {
          loadDirectory();
        }
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

  List<Map<String, dynamic>> currentChannelMembers = [];

  Future<void> selectChannel(String channelId) async {
    String resolvedChannelId = channelId;

    // Selecting a user in DM list should resolve to a concrete DM channel first.
    final selectedUser = users.where((u) => u.id == channelId).firstOrNull;
    if (selectedUser != null) {
      final dmChannelId = await messagingService.createDMChannel(selectedUser.id);
      if (dmChannelId != null && dmChannelId.isNotEmpty) {
        resolvedChannelId = dmChannelId;
      }
    }

    currentChannelId = resolvedChannelId;
    currentChannelMembers = []; // Reset
    
    // Safety check: if choosing a user for DM that isn't yet in history but is in users list,
    // this handles the initial view.
    final known = channels.any((c) => c['id'] == resolvedChannelId) || users.any((u) => u.id == resolvedChannelId);
    if (!known) {
      await loadDirectory();
    }

    if (!messageHistory.containsKey(resolvedChannelId)) {
      final history = await messagingService.getHistory(resolvedChannelId);
      messageHistory[resolvedChannelId] = history;
    }

    // Load members if it's a channel (not a DM)
    if (channels.any((c) => c['id'] == resolvedChannelId)) {
      currentChannelMembers = await directoryService.getChannelMembers(resolvedChannelId);
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
