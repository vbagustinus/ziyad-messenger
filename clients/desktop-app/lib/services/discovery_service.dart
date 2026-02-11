import 'dart:io';
import 'dart:convert';
import 'dart:async';

class DiscoveryService {
  RawDatagramSocket? _socket;
  final int port = 5353;
  final String multicastAddress = '224.0.0.251';
  final StreamController<Map<String, dynamic>> _peerController = StreamController.broadcast();

  Stream<Map<String, dynamic>> get peerStream => _peerController.stream;

  Future<void> start() async {
    try {
      _socket = await RawDatagramSocket.bind(InternetAddress.anyIPv4, port);
      _socket!.multicastLoopback = false;
      _socket!.joinMulticast(InternetAddress(multicastAddress));

      _socket!.listen((RawSocketEvent event) {
        if (event == RawSocketEvent.read) {
          final datagram = _socket!.receive();
          if (datagram != null) {
            try {
              final String message = utf8.decode(datagram.data);
              final data = jsonDecode(message);
              _peerController.add(data);
            } catch (e) {
              print('Error parsing UDP packet: $e');
            }
          }
        }
      });
      print('Discovery Service listening on port $port');
    } catch (e) {
      print('Error starting Discovery Service: $e');
    }
  }

  void stop() {
    _socket?.close();
    _peerController.close();
  }
}
