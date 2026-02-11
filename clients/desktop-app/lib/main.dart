import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'config/app_config.dart';
import 'services/auth_service.dart';
import 'ui/login_screen.dart';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    // We'll wrap with MultiProvider so we can add ChatProvider later after login
    return MultiProvider(
      providers: [
        Provider<AuthService>(create: (_) => AuthService(baseUrl: AppConfig.authBaseUrl)),
      ],
      child: MaterialApp(
        title: 'Ziyad Messenger',
        debugShowCheckedModeBanner: false,
        theme: ThemeData(
          colorScheme: ColorScheme.fromSeed(
            seedColor: const Color(0xFF6366F1), // Indigo/Slack-ish
            primary: const Color(0xFF6366F1),
          ),
          useMaterial3: true,
          fontFamily: 'Inter', // If you have the font, else default
        ),
        initialRoute: '/',
        routes: {
          '/': (context) => LoginScreen(authService: context.read<AuthService>()),
        },
      ),
    );
  }
}
