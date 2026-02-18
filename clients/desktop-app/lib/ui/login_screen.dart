import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/auth_service.dart';
import '../services/messaging_service.dart';
import '../services/directory_service.dart';
import '../providers/chat_provider.dart';
import '../config/app_config.dart';
import 'chat_screen.dart';
import 'connection_test_screen.dart';

class LoginScreen extends StatefulWidget {
  final AuthService authService;

  const LoginScreen({super.key, required this.authService});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _isLoading = false;
  bool _isRegister = false;

  void _submit() async {
    final username = _usernameController.text.trim();
    final password = _passwordController.text;
    if (username.isEmpty || password.isEmpty) {
      _showError('Enter username and password');
      return;
    }

    setState(() => _isLoading = true);
    
    bool? success;
    if (_isRegister) {
      final id = await widget.authService.register(username, password);
      if (id != null) {
        success = await widget.authService.login(username, password);
      }
    } else {
      success = await widget.authService.login(username, password);
    }

    setState(() => _isLoading = false);

    if (success == true && mounted) {
      final user = widget.authService.currentUser!;
      
      // Initialize services with the authenticated user data
      final msgService = MessagingService(
        httpBaseUrl: AppConfig.messagingBaseUrl,
        wsBaseUrl: AppConfig.messagingBaseUrl.replaceAll('http://', 'ws://'),
        token: user.token ?? '',
        userId: user.id,
      );
      
      final dirService = DirectoryService(
        adminBaseUrl: AppConfig.adminApiBaseUrl,
        token: user.token ?? '',
      );

      Navigator.pushReplacement(
        context,
        MaterialPageRoute(
          builder: (_) => ChangeNotifierProvider(
            create: (_) => ChatProvider(
              messagingService: msgService,
              directoryService: dirService,
            ),
            child: const ChatScreen(),
          ),
        ),
      );
    } else if (mounted) {
      _showError(_isRegister ? 'Registration failed' : 'Login failed');
    }
  }

  void _showError(String msg) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(msg),
      backgroundColor: Colors.redAccent,
      behavior: SnackBarBehavior.floating,
    ));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFFF3F4F6),
      body: Row(
        children: [
          // Left side: Branding / Illustration
          Expanded(
            flex: 4,
            child: Container(
              color: const Color(0xFF6366F1),
              child: const Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.chat_bubble_rounded, size: 100, color: Colors.white),
                  SizedBox(height: 24),
                  Text(
                    'Ziyad Messenger',
                    style: TextStyle(color: Colors.white, fontSize: 32, fontWeight: FontWeight.bold),
                  ),
                  SizedBox(height: 8),
                  Text(
                    'Secure. Real-time. Ceria.',
                    style: TextStyle(color: Colors.white70, fontSize: 18),
                  ),
                ],
              ),
            ),
          ),
          // Right side: Login Form
          Expanded(
            flex: 3,
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 48),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    _isRegister ? 'Join Ziyad' : 'Welcome Back',
                    style: const TextStyle(fontSize: 28, fontWeight: FontWeight.bold, color: Color(0xFF1F2937)),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    _isRegister ? 'Create your account to start chatting' : 'Sign in to your account',
                    style: const TextStyle(color: Color(0xFF6B7280)),
                  ),
                  const SizedBox(height: 32),
                  _buildLabel('Username'),
                  _buildTextField(_usernameController, 'e.g. ziyadbooks', false),
                  const SizedBox(height: 16),
                  _buildLabel('Password'),
                  _buildTextField(_passwordController, '••••••••', true),
                  const SizedBox(height: 24),
                  SizedBox(
                    width: double.infinity,
                    height: 50,
                    child: ElevatedButton(
                      onPressed: _isLoading ? null : _submit,
                      style: ElevatedButton.styleFrom(
                        backgroundColor: const Color(0xFF6366F1),
                        foregroundColor: Colors.white,
                        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
                      ),
                      child: _isLoading 
                        ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2))
                        : Text(_isRegister ? 'Register Now' : 'Sign In', style: const TextStyle(fontWeight: FontWeight.bold)),
                    ),
                  ),
                  const SizedBox(height: 16),
                  Center(
                    child: TextButton(
                      onPressed: () => setState(() => _isRegister = !_isRegister),
                      child: Text(
                        _isRegister ? 'Already have an account? Sign in' : 'Don\'t have an account? Register',
                        style: const TextStyle(color: Color(0xFF6366F1)),
                      ),
                    ),
                  )
                ],
              ),
            ),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton.small(
        onPressed: () {
          Navigator.push(
            context,
            MaterialPageRoute(
              builder: (_) => const ConnectionTestScreen(),
            ),
          );
        },
        backgroundColor: Colors.grey[300],
        child: const Icon(Icons.network_check, color: Colors.black54),
      ),
    );
  }

  Widget _buildLabel(String text) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Text(text, style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14)),
    );
  }

  Widget _buildTextField(TextEditingController controller, String hint, bool obscure) {
    return TextField(
      controller: controller,
      obscureText: obscure,
      decoration: InputDecoration(
        hintText: hint,
        hintStyle: const TextStyle(color: Colors.grey, fontSize: 14),
        filled: true,
        fillColor: Colors.white,
        border: OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: const BorderSide(color: Color(0xFFE5E7EB))),
        enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: const BorderSide(color: Color(0xFFE5E7EB))),
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      ),
    );
  }
}
