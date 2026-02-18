import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';
import '../models/user.dart';

class AuthService {
  final String baseUrl;
  User? _currentUser;

  AuthService({required this.baseUrl});

  User? get currentUser => _currentUser;

  /// Register a new user. Call this before login if the account doesn't exist.
  Future<String?> register(String username, String password, {String role = 'member', String? fullName}) async {
    try {
      final response = await http.post(
        Uri.parse('$baseUrl/register'),
        body: jsonEncode({
          'username': username,
          'full_name': fullName ?? username,
          'password': password,
          'role': role,
        }),
        headers: {'Content-Type': 'application/json'},
      );
      if (response.statusCode == 201) {
        final data = jsonDecode(response.body) as Map<String, dynamic>;
        return data['id'] as String?;
      }
      return null;
    } catch (e) {
      print('Register error: $e');
      return null;
    }
  }

  Future<bool> login(String username, String password) async {
    try {
      final response = await http.post(
        Uri.parse('$baseUrl/login'),
        body: jsonEncode({'username': username, 'password': password}),
        headers: {'Content-Type': 'application/json'},
      );

      if (response.statusCode == 200) {
        final body = response.body;
        if (body.isEmpty) {
          print('Login error: empty response body');
          return false;
        }
        final data = jsonDecode(body) as Map<String, dynamic>;
        final token = data['token'] as String?;
        final userId = data['user_id'] as String? ?? '';
        final role = data['role'] as String? ?? 'user';

        _currentUser = User(
          id: userId,
          username: username,
          role: role,
          token: token ?? '',
        );

        final prefs = await SharedPreferences.getInstance();
        await prefs.setString('jwt_token', token ?? '');
        return true;
      }
      // Log for debugging (e.g. 401 = wrong password, 404 = wrong URL)
      print('Login failed: status=${response.statusCode} body=${response.body}');
      return false;
    } catch (e) {
      print('Login error: $e');
      return false;
    }
  }

  Future<void> logout() async {
    _currentUser = null;
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove('jwt_token');
  }

  Future<String?> getToken() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString('jwt_token');
  }
}
