
import 'dart:async';
import 'dart:math';

import 'package:flutter/widgets.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:viam_sdk/viam_sdk.dart';
import 'api.dart';

Future<void> main() async {
  print('Testing videostore client');
  WidgetsFlutterBinding.ensureInitialized();
  try {
    // Need to hit custom registration hook since this is a custom API
    ensureVideostoreRegistered();

    // Load environment variables from .env file
    await dotenv.load(); // loads from package root
    final host = dotenv.maybeGet('VIAM_HOST') ?? '';
    final apiKeyID = dotenv.maybeGet('VIAM_API_KEY_ID') ?? '';
    final apiKey = dotenv.maybeGet('VIAM_API_KEY') ?? '';
    if ([host, apiKeyID, apiKey].any((v) => v.isEmpty)) {
      print('Missing required env vars (VIAM_HOST / VIAM_API_KEY_ID / VIAM_API_KEY)');
      return;
    }

    final machine = await RobotClient.atAddress(
      host,
      RobotClientOptions.withApiKey(apiKeyID, apiKey),
    );
    print('resourceNames: ${machine.resourceNames}');

    final serviceName = dotenv.maybeGet('VIAM_SERVICE_NAME') ?? '';
    if (serviceName.isEmpty) {
      print('Missing required env var VIAM_SERVICE_NAME');
      return;
    }
    print('serviceName: $serviceName');
    final rn = VideoStore.subtype.getResourceName(serviceName);
    if (!machine.resourceNames.contains(rn)) {
      print('Resource $rn not found on robot.');
      return;
    }
    final videostore = VideoStore.fromRobot(machine, serviceName);

    final now = DateTime.now().toUtc();
    String fmtYmdHms(DateTime d) {
      final y = d.year.toString().padLeft(4, '0');
      final m = d.month.toString().padLeft(2, '0');
      final day = d.day.toString().padLeft(2, '0');
      final hh = d.hour.toString().padLeft(2, '0');
      final mm = d.minute.toString().padLeft(2, '0');
      final ss = d.second.toString().padLeft(2, '0');
      return '$y-$m-$day\_${hh}-$mm-${ss}Z';
    }
    final to = fmtYmdHms(now.subtract(const Duration(seconds: 40)));
    final from = fmtYmdHms(now.subtract(const Duration(seconds: 50)));

    // Test streaming fetch
    print('Streaming fetch from $from to $to');
    int totalStreamedBytes = 0;
    final sw = Stopwatch()..start();
    bool done = false;
    final stream = videostore.fetchStream(from, to);
    late StreamSubscription<List<int>> sub;
    sub = stream.listen(
        (chunk) {
          print('Main: got chunk length=${chunk.length}');
          totalStreamedBytes += chunk.length;
        },
        onError: (e) {
          print('Stream error: $e');
          if (!done) {
            done = true;
            sw.stop();
            sub.cancel();
          }
        },
        onDone: () {
          if (!done) {
            done = true;
            sw.stop();
          }
          print("Stream done. Total bytes: $totalStreamedBytes in ${sw.elapsedMilliseconds} ms");
        }
    );

    // Test unary fetch
    print('Unary fetch from $from to $to');
    final result = await videostore.fetch(from, to);
    print('Unary fetch complete, Bytes: ${result.video_data.length}');

    // Gracefully shutdown flutter app
    await machine.close();
    print('Test complete, exiting');

  } catch (e, st) {
    print('Unhandled exception: $e');
    print('Stack trace:\n$st');
  }
}
