import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/user.dart';

class DirectoryService {
  final String adminBaseUrl;
  final String token;

  DirectoryService({required this.adminBaseUrl, required this.token});

  Future<List<User>> getUsers() async {
    try {
      final response = await http.get(
        Uri.parse('$adminBaseUrl/public/users'),
        headers: {
          'Content-Type': 'application/json',
        },
      );

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        final List usersData = data['users'] ?? [];
        return usersData.map((u) => User(
          id: u['id'],
          username: u['username'],
          role: u['role_id'] ?? 'user',
          token: '', // Token not needed for directory users
        )).toList();
      }
      return [];
    } catch (e) {
      print('Error fetching users: $e');
      return [];
    }
  }

  Future<List<Map<String, dynamic>>> getChannels() async {
    try {
      final response = await http.get(
        Uri.parse('$adminBaseUrl/public/channels'),
        headers: {
          'Content-Type': 'application/json',
        },
      );

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        return List<Map<String, dynamic>>.from(data['channels'] ?? []);
      }
      return [];
    } catch (e) {
      print('Error fetching channels: $e');
      return [];
    }
  }

  Future<List<Map<String, dynamic>>> getChannelMembers(String channelId) async {
    try {
      final response = await http.get(
        Uri.parse('$adminBaseUrl/admin/channels/$channelId/members'),
        headers: {
          'Content-Type': 'application/json',
          if (token.isNotEmpty) 'Authorization': 'Bearer $token',
        },
      );

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        return List<Map<String, dynamic>>.from(data['members'] ?? []);
      }
      return [];
    } catch (e) {
      print('Error fetching channel members: $e');
      return [];
    }
  }
}
