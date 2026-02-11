import 'dart:convert';
import 'dart:io';
import 'package:http/http.dart' as http;
import 'package:path/path.dart' as path;

class FileService {
  final String baseUrl;

  FileService({required this.baseUrl});

  Future<Map<String, dynamic>?> uploadFile(File file) async {
    try {
      final request = http.MultipartRequest('POST', Uri.parse('$baseUrl/upload'));
      request.files.add(await http.MultipartFile.fromPath('file', file.path));

      final streamedResponse = await request.send();
      final response = await http.Response.fromStream(streamedResponse);

      if (response.statusCode == 201) {
        return jsonDecode(response.body);
      }
      print('Upload failed: ${response.statusCode}');
      return null;
    } catch (e) {
      print('Upload error: $e');
      return null;
    }
  }

  String getDownloadUrl(String fileId) {
    return '$baseUrl/download?id=$fileId';
  }
}
