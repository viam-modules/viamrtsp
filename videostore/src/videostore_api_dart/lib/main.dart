
import 'package:flutter/widgets.dart';
import 'package:viam_sdk/viam_sdk.dart';
import 'api.dart';

// Future<void> main() async {
//   print('Testing videostore client');
//   const host = 'framework-1-main.wcfb1lr0dn.viam.cloud'; 
//   const apiKeyID = '093a5315-d9de-4689-9d2b-e9b95dd9ab84'; 
//   const apiKey = 'b463sf1xqqaeiwpjnhtyzgxu1qi7rgu9';
//   final machine = await RobotClient.atAddress(
//     host,
//     RobotClientOptions.withApiKey(apiKeyID, apiKey),
//   );
//   print(machine.resourceNames);
// }

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  try {
    // await Future.delayed(const Duration(seconds: 5));
    ensureVideostoreRegistered(); // <--- ensure registry entry exists
    print('Testing videostore client');
    const host = 'framework-1-main.wcfb1lr0dn.viam.cloud';
    const apiKeyID = '093a5315-d9de-4689-9d2b-e9b95dd9ab84';
    const apiKey = 'b463sf1xqqaeiwpjnhtyzgxu1qi7rgu9';
    // final dialOpts = DialOptions()
    //   ..webRtcOptions = (DialWebRtcOptions()..disable = true);

    // final robotOpts = RobotClientOptions.withApiKeyAndDialOptions(apiKeyID, apiKey, dialOpts);
    // final machine = await RobotClient.atAddress(
    //   host,
    //   robotOpts,
    // );
    final machine = await RobotClient.atAddress(
      host,
      RobotClientOptions.withApiKey(apiKeyID, apiKey),
    );

    print('resourceNames: ${machine.resourceNames}');
    final rn = VideoStore.subtype.getResourceName('vs-1');
    print('expected ResourceName: $rn');
    if (!machine.resourceNames.contains(rn)) {
      print('Resource not found on robot; available resources listed above.');
      return;
    }
    final videostore = VideoStore.fromRobot(machine, 'vs-1');
    print('Fetching video data...');
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
    print('Fetching from $from to $to');
    // Test streaming fetch
    final stream = videostore.fetchStream(from, to);
    final sub = stream.listen(
        (chunk) => print('Main: got chunk length=${chunk.length}'),
        onError: (e) => print('stream error: $e'),
        onDone: () => print('stream done'),
    );
    Future.delayed(const Duration(seconds: 30), () async {
      await sub.cancel();
      print('Subscription cancelled by timeout');
    });
    // Test unary fetch
    final result = await videostore.fetch(from, to);
    print('Fetched video data of size: ${result.video_data.length}');
    print('Fetch complete.');
  } catch (e, st) {
    print('Unhandled exception: $e');
    print('Stack trace:\n$st');
  }
}
