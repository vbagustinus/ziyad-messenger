import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';
import '../config/app_config.dart';

class ConnectionTestScreen extends StatefulWidget {
  const ConnectionTestScreen({super.key});

  @override
  State<ConnectionTestScreen> createState() => _ConnectionTestScreenState();
}

class _ConnectionTestScreenState extends State<ConnectionTestScreen> {
  final Map<String, String?> _results = {};
  bool _testing = false;

  Future<void> _testConnection(String name, String url) async {
    try {
      final response = await http.get(Uri.parse(url)).timeout(
        const Duration(seconds: 3),
      );
      setState(() {
        _results[name] = response.statusCode == 200
            ? '✓ OK (${response.statusCode})'
            : '✗ Failed (${response.statusCode})';
      });
    } catch (e) {
      setState(() {
        _results[name] = '✗ Error: ${e.toString()}';
      });
    }
  }

  Future<void> _testLogin() async {
    try {
      final response = await http.post(
        Uri.parse('${AppConfig.authBaseUrl}/login'),
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({
          'username': 'admin',
          'password': 'password',
        }),
      ).timeout(const Duration(seconds: 5));

      setState(() {
        if (response.statusCode == 200) {
          final data = jsonDecode(response.body);
          _results['Login Test'] = '✓ Success\\nToken: ${data['token']?.substring(0, 20)}...\\nRole: ${data['role']}';
        } else {
          _results['Login Test'] = '✗ Failed (${response.statusCode})\\n${response.body}';
        }
      });
    } catch (e) {
      setState(() {
        _results['Login Test'] = '✗ Error: ${e.toString()}';
      });
    }
  }

  Future<void> _runAllTests() async {
    setState(() {
      _testing = true;
      _results.clear();
    });

    await _testConnection('Auth Service', '${AppConfig.authBaseUrl}/health');
    await _testConnection('Messaging Service', '${AppConfig.messagingBaseUrl}/health');
    await _testConnection('Presence Service', '${AppConfig.presenceBaseUrl}/health');
    await _testConnection('File Transfer Service', '${AppConfig.fileTransferBaseUrl}/health');
    await _testConnection('Admin API', '${AppConfig.adminApiBaseUrl}/health');
    await _testLogin();

    setState(() {
      _testing = false;
    });
  }

  @override
  void initState() {
    super.initState();
    _runAllTests();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Connection Test'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: _testing ? null : _runAllTests,
          ),
        ],
      ),
      body: _testing
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text(
                          'Configuration',
                          style: TextStyle(
                            fontSize: 18,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                        const SizedBox(height: 8),
                        _buildConfigRow('Auth', AppConfig.authBaseUrl),
                        _buildConfigRow('Messaging', AppConfig.messagingBaseUrl),
                        _buildConfigRow('Presence', AppConfig.presenceBaseUrl),
                        _buildConfigRow('File Transfer', AppConfig.fileTransferBaseUrl),
                        _buildConfigRow('Admin API', AppConfig.adminApiBaseUrl),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 16),
                const Text(
                  'Test Results',
                  style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                ),
                const SizedBox(height: 8),
                ..._results.entries.map((entry) => Card(
                      child: ListTile(
                        title: Text(entry.key),
                        subtitle: Text(
                          entry.value ?? 'Testing...',
                          style: TextStyle(
                            color: entry.value?.startsWith('✓') == true
                                ? Colors.green
                                : Colors.red,
                          ),
                        ),
                        leading: Icon(
                          entry.value?.startsWith('✓') == true
                              ? Icons.check_circle
                              : Icons.error,
                          color: entry.value?.startsWith('✓') == true
                              ? Colors.green
                              : Colors.red,
                        ),
                      ),
                    )),
              ],
            ),
    );
  }

  Widget _buildConfigRow(String label, String url) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 100,
            child: Text(
              '$label:',
              style: const TextStyle(fontWeight: FontWeight.w500),
            ),
          ),
          Expanded(
            child: SelectableText(
              url,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
            ),
          ),
        ],
      ),
    );
  }
}
