import 'package:grpc/grpc_connection_interface.dart';
import 'package:viam_sdk/protos/app/app.dart';
import 'grpc/src/proto/videostore.pb.dart';
import 'grpc/src/proto/videostore.pbgrpc.dart';
import 'package:viam_sdk/viam_sdk.dart';
import 'dart:async';

// Need to fire off this before fetching videostore resources
// This is done automatically with built-in SDK resources
void ensureVideostoreRegistered() {
  try {
    if (!Registry.instance.subtypes.containsKey(VideoStore.subtype)) {
      Registry.instance.registerSubtype(
        ResourceRegistration<VideoStore>(
          VideoStore.subtype,
          (String name, ClientChannelBase channel) => VideostoreClient(name, channel),
        ),
      );
    }
  } catch (_) {
    // ignore if already registered or registration fails
    print('Error registering VideoStore subtype, possibly already registered.');
  }
}
/*
    "api": "viam-modules:service:videostore",
    "model": "viam:viamrtsp:video-store",
*/
class FetchResult {
    List<int> video_data;
    FetchResult({
      required this.video_data,
    });
}

class SaveResult {
    String filename;
    SaveResult({
      required this.filename,
    });
}

// create wrapper class for grpc client and server
abstract class VideoStore extends Resource {
    static const Subtype subtype = Subtype('viam-modules', 'service', 'videostore');
    Future<FetchResult> fetch(String from, String to);
    Future<SaveResult> save(String from, String to);
    Stream<List<int>> fetchStream(String from, String to);
    static ResourceName getResourceName(String name) {
        return VideoStore.subtype.getResourceName(name);
    }
    static VideoStore fromRobot(RobotClient robot, String name) {
        return robot.getResource(VideoStore.getResourceName(name));
    }
}

// create client class that extends ViamClient and implements VideoStore
class VideostoreClient extends VideoStore with RPCDebugLoggerMixin implements ResourceRPCClient {
    @override
    final String name;

    @override
    ClientChannelBase channel;

    @override
    videostoreServiceClient get client => videostoreServiceClient(channel);
  
    VideostoreClient(this.name, this.channel);

    @override
    Future<FetchResult> fetch(String from, String to) async {
        final request = FetchRequest()
            ..name = name
            ..from = from
            ..to = to;
        print("calling fetch with from: $from, to: $to, name: $name");
        final response = await client.fetch(request);
        return FetchResult(video_data: response.videoData);
    }

    @override
    Future<SaveResult> save(String from, String to) async {
        final request = SaveRequest()
            ..name = name
            ..from = from
            ..to = to;
        final response = await client.save(request);
        return SaveResult(filename: response.filename);
    }

    @override
    Stream<List<int>> fetchStream(String from, String to) {
        print("fetchStream called with from: $from, to: $to, name: $name");
        final request = FetchStreamRequest()
            ..name = name
            ..from = from
            ..to = to;

        final response = client.fetchStream(request);
        final mapped = response.map((resp) {
            final len = resp.videoData.length;
            print('fetchStream: received chunk size=$len');
            return resp.videoData;
        });
        return mapped;
    }
}