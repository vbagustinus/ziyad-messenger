import 'dart:io';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:file_picker/file_picker.dart';
import 'package:intl/intl.dart';
import '../providers/chat_provider.dart';
import '../services/auth_service.dart';
import '../services/file_service.dart';
import '../models/message.dart';
import '../config/app_config.dart';

class ChatScreen extends StatefulWidget {
  const ChatScreen({super.key});

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final TextEditingController _messageController = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  late FileService _fileService;

  @override
  void initState() {
    super.initState();
    _fileService = FileService(baseUrl: AppConfig.fileTransferBaseUrl);
    // Initial data load
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<ChatProvider>().loadDirectory();
      context.read<ChatProvider>().messagingService.connect();
    });
  }

  void _scrollToBottom() {
    if (_scrollController.hasClients) {
      _scrollController.animateTo(
        _scrollController.position.maxScrollExtent,
        duration: const Duration(milliseconds: 300),
        curve: Curves.easeOut,
      );
    }
  }

  Future<void> _pickAndUploadFile() async {
    final result = await FilePicker.platform.pickFiles();
    if (result != null && result.files.single.path != null) {
      final file = File(result.files.single.path!);
      final res = await _fileService.uploadFile(file);
      if (res != null) {
        final fileId = res['file_id'];
        final provider = context.read<ChatProvider>();
        // Send a message with file ID
        await provider.messagingService.sendMessage(
          provider.currentChannelId!, 
          'FILE:$fileId:${result.files.single.name}', 
          MessageType.file
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<ChatProvider>();
    final auth = context.read<AuthService>();
    final currentUserId = auth.currentUser?.id;

    return Scaffold(
      body: Row(
        children: [
          // Sidebar
          Container(
            width: 260,
            color: const Color(0xFF3F0E40), // Classic Slack Purple
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const SizedBox(height: 50),
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  child: Row(
                    children: [
                      Container(
                        padding: const EdgeInsets.all(8),
                        decoration: BoxDecoration(
                          color: Colors.white24,
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: const Icon(Icons.bubble_chart, color: Colors.white, size: 20),
                      ),
                      const SizedBox(width: 12),
                      const Text(
                        'Ziyad Messenger',
                        style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 18),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 24),
                _buildSidebarSection(
                  title: 'Channels',
                  items: provider.channels,
                  onSelect: (id) => provider.selectChannel(id),
                  currentId: provider.currentChannelId,
                  icon: Icons.tag,
                ),
                _buildSidebarSection(
                  title: 'Direct Messages',
                  items: provider.users.where((u) => u.id != currentUserId).map((u) => {'id': u.id, 'name': u.username}).toList(),
                  onSelect: (id) => provider.selectChannel(id),
                  currentId: provider.currentChannelId,
                  icon: Icons.circle,
                  iconColor: Colors.green,
                ),
                const Spacer(),
                Container(
                  padding: const EdgeInsets.all(16),
                  color: Colors.black12,
                  child: Row(
                    children: [
                      CircleAvatar(
                        backgroundColor: Colors.white24,
                        child: Text(auth.currentUser?.username[0].toUpperCase() ?? '?', style: const TextStyle(color: Colors.white)),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          auth.currentUser?.username ?? '',
                          style: const TextStyle(color: Colors.white70, fontWeight: FontWeight.w500),
                        ),
                      ),
                      IconButton(
                        icon: const Icon(Icons.logout, color: Colors.white54, size: 20),
                        onPressed: () {
                          auth.logout();
                          Navigator.pushReplacementNamed(context, '/');
                        },
                      )
                    ],
                  ),
                ),
              ],
            ),
          ),
          
          // Chat Area
          Expanded(
            child: Container(
              color: Colors.white,
              child: provider.currentChannelId == null
                  ? const Center(child: Text('Select a channel or user to start chatting', style: TextStyle(color: Colors.black54, fontSize: 16)))
                  : Column(
                      children: [
                        // Header
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
                          decoration: const BoxDecoration(
                            border: Border(bottom: BorderSide(color: Colors.black12)),
                          ),
                          child: Row(
                            children: [
                              Text(
                                provider.getChannelName(provider.currentChannelId!),
                                style: const TextStyle(fontWeight: FontWeight.w900, fontSize: 18),
                              ),
                              const Spacer(),
                              const Icon(Icons.info_outline, color: Colors.black38, size: 20),
                            ],
                          ),
                        ),
                        
                        // Messages
                        Expanded(
                          child: ListView.builder(
                            controller: _scrollController,
                            padding: const EdgeInsets.all(20),
                            itemCount: provider.messageHistory[provider.currentChannelId!]?.length ?? 0,
                            itemBuilder: (context, index) {
                              final msg = provider.messageHistory[provider.currentChannelId!]![index];
                              return _buildMessageItem(msg, currentUserId);
                            },
                          ),
                        ),
                        
                        // Input
                        Container(
                          padding: const EdgeInsets.all(16),
                          child: Container(
                            decoration: BoxDecoration(
                              border: Border.all(color: Colors.black12),
                              borderRadius: BorderRadius.circular(8),
                            ),
                            child: Column(
                              children: [
                                Row(
                                  children: [
                                    IconButton(
                                      icon: const Icon(Icons.add, color: Colors.black54),
                                      onPressed: _pickAndUploadFile,
                                    ),
                                    const Icon(Icons.sentiment_satisfied_alt_outlined, color: Colors.black54),
                                    const SizedBox(width: 8),
                                    Expanded(
                                      child: TextField(
                                        controller: _messageController,
                                        decoration: InputDecoration(
                                          hintText: 'Message ${provider.getChannelName(provider.currentChannelId!)}',
                                          border: InputBorder.none,
                                          contentPadding: const EdgeInsets.symmetric(vertical: 0),
                                        ),
                                        onSubmitted: (val) {
                                          if (val.trim().isNotEmpty) {
                                            provider.sendText(val.trim());
                                            _messageController.clear();
                                            _scrollToBottom();
                                          }
                                        },
                                      ),
                                    ),
                                    IconButton(
                                      icon: const Icon(Icons.send, color: Color(0xFF007A5A)), // Slack Green
                                      onPressed: () {
                                        if (_messageController.text.trim().isNotEmpty) {
                                          provider.sendText(_messageController.text.trim());
                                          _messageController.clear();
                                          _scrollToBottom();
                                        }
                                      },
                                    ),
                                  ],
                                ),
                                Container(
                                  color: const Color(0xFFF8F8F8),
                                  height: 36,
                                  child: const Row(
                                    children: [
                                      SizedBox(width: 48),
                                      Icon(Icons.format_bold, size: 18, color: Colors.black45),
                                      SizedBox(width: 16),
                                      Icon(Icons.format_italic, size: 18, color: Colors.black45),
                                      SizedBox(width: 16),
                                      Icon(Icons.link, size: 18, color: Colors.black45),
                                      SizedBox(width: 16),
                                      Icon(Icons.format_list_bulleted, size: 18, color: Colors.black45),
                                    ],
                                  ),
                                )
                              ],
                            ),
                          ),
                        ),
                      ],
                    ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSidebarSection({
    required String title,
    required List<dynamic> items,
    required Function(String) onSelect,
    String? currentId,
    IconData? icon,
    Color? iconColor,
  }) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          child: Text(
            title,
            style: const TextStyle(color: Colors.white54, fontWeight: FontWeight.bold, fontSize: 13),
          ),
        ),
        ...items.map((item) {
          final isSelected = item['id'] == currentId;
          return Material(
            color: isSelected ? const Color(0xFF1164A3) : Colors.transparent,
            child: InkWell(
              onTap: () => onSelect(item['id']),
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
                child: Row(
                  children: [
                    Icon(icon, color: isSelected ? Colors.white : (iconColor ?? Colors.white38), size: 16),
                    const SizedBox(width: 12),
                    Text(
                      item['name'],
                      style: TextStyle(
                        color: isSelected ? Colors.white : Colors.white70,
                        fontSize: 15,
                        fontWeight: isSelected ? FontWeight.bold : FontWeight.normal,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          );
        }).toList(),
        const SizedBox(height: 16),
      ],
    );
  }

  Widget _buildMessageItem(Message msg, String? currentUserId) {
    // In a real Slack app, messages from the same user are clustered.
    // For now, let's keep it simple.
    final timeStr = DateFormat('h:mm a').format(msg.timestamp);
    final isFile = msg.content.startsWith('FILE:');
    
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 36,
            height: 36,
            decoration: BoxDecoration(
              color: Colors.blueGrey[100],
              borderRadius: BorderRadius.circular(4),
            ),
            child: Center(child: Text(msg.senderId[0].toUpperCase())),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Text(msg.senderId == currentUserId ? 'Me' : msg.senderId.split('-').first, style: const TextStyle(fontWeight: FontWeight.bold)),
                    const SizedBox(width: 8),
                    Text(timeStr, style: const TextStyle(color: Colors.black38, fontSize: 11)),
                  ],
                ),
                isFile 
                  ? _buildFileContent(msg.content)
                  : Text(msg.content, style: const TextStyle(fontSize: 15, height: 1.4)),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildFileContent(String content) {
    final parts = content.split(':');
    if (parts.length < 3) return const Text('Invalid file');
    final fileId = parts[1];
    final fileName = parts[2];

    return Container(
      margin: const EdgeInsets.only(top: 8),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        border: Border.all(color: Colors.black12),
        borderRadius: BorderRadius.circular(8),
        color: const Color(0xFFF8F8F8),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(Icons.insert_drive_file, color: Colors.blueAccent),
          const SizedBox(width: 12),
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(fileName, style: const TextStyle(fontWeight: FontWeight.w600)),
              const Text('Binary File', style: TextStyle(fontSize: 12, color: Colors.black38)),
            ],
          ),
          const SizedBox(width: 24),
          IconButton(
            icon: const Icon(Icons.download, color: Colors.black54),
            onPressed: () {
              // In a real app, use url_launcher or custom download logic
              print('Downloading $fileId');
            },
          )
        ],
      ),
    );
  }
}
