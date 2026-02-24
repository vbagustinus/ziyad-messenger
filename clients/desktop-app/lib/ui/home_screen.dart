import 'package:flutter/material.dart';
import '../config/app_config.dart';
import '../services/auth_service.dart';
import '../services/messaging_service.dart';
import 'chat_screen.dart';

class HomeScreen extends StatelessWidget {
  final AuthService authService;

  const HomeScreen({super.key, required this.authService});

  @override
  Widget build(BuildContext context) {
    // Mock Channels
    final channels = [
      {'id': 'general', 'name': 'General'},
      {'id': 'random', 'name': 'Random'},
      {'id': 'dev', 'name': 'Development'},
    ];

    // Mock DMs
    final directMessages = [
      {'id': 'user1', 'name': 'Alice'},
      {'id': 'user2', 'name': 'Bob'},
    ];

    return DefaultTabController(
      length: 2,
      child: Scaffold(
        appBar: AppBar(
          title: const Text('Secure LAN Chat'),
          actions: [
            IconButton(
              icon: const Icon(Icons.logout),
              onPressed: () {
                authService.logout();
                Navigator.pushReplacementNamed(context, '/');
              },
            ),
          ],
          bottom: const TabBar(
            tabs: [
              Tab(text: 'Channels', icon: Icon(Icons.tag)),
              Tab(text: 'DMs', icon: Icon(Icons.person)),
            ],
          ),
        ),
        body: TabBarView(
          children: [
            _buildList(context, channels),
            _buildList(context, directMessages),
          ],
        ),
      ),
    );
  }

  Widget _buildList(BuildContext context, List<Map<String, String>> items) {
    return ListView.builder(
      itemCount: items.length,
      itemBuilder: (context, index) {
        final item = items[index];
        return ListTile(
          leading: Icon(item['id']!.startsWith('user') ? Icons.circle : Icons.tag),
          title: Text(item['name']!),
          onTap: () {
            Navigator.push(
              context,
              MaterialPageRoute(
                builder: (_) => ChatScreen(
                  currentUser: authService.currentUser!,
                  channelId: item['id']!,
                  channelName: item['name']!,
                  messagingService: MessagingService(
                    httpBaseUrl: AppConfig.messagingBaseUrl,
                    wsBaseUrl: AppConfig.wsMessagingBaseUrl,
                    token: authService.currentUser!.token ?? '',
                  ),
                ),
              ),
            );
          },
        );
      },
    );
  }
}
