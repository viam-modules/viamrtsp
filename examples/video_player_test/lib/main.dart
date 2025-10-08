import 'package:flutter/material.dart';
import 'package:viam_sdk/viam_sdk.dart';
import 'package:videostore_api/videostore_api.dart';
import 'package:video_player/video_player.dart';
import 'dart:async';
import 'dart:io';

void main() {
  WidgetsFlutterBinding.ensureInitialized(); // Ensure platform channels ready
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  
  const MyApp({super.key});

  // This widget is the root of your application.
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Flutter Demo',
      theme: ThemeData(
        // This is the theme of your application.
        //
        // TRY THIS: Try running your application with "flutter run". You'll see
        // the application has a purple toolbar. Then, without quitting the app,
        // try changing the seedColor in the colorScheme below to Colors.green
        // and then invoke "hot reload" (save your changes or press the "hot
        // reload" button in a Flutter-supported IDE, or press "r" if you used
        // the command line to start the app).
        //
        // Notice that the counter didn't reset back to zero; the application
        // state is not lost during the reload. To reset the state, use hot
        // restart instead.
        //
        // This works for code too, not just values: Most code changes can be
        // tested with just a hot reload.
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.deepPurple),
      ),
      home: const MyHomePage(title: 'Flutter Demo Home Page'),
    );
  }
}

class MyHomePage extends StatefulWidget {
  const MyHomePage({super.key, required this.title});

  // This widget is the home page of your application. It is stateful, meaning
  // that it has a State object (defined below) that contains fields that affect
  // how it looks.

  // This class is the configuration for the state. It holds the values (in this
  // case the title) provided by the parent (in this case the App widget) and
  // used by the build method of the State. Fields in a Widget subclass are
  // always marked "final".

  final String title;

  @override
  State<MyHomePage> createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  int _counter = 0;
  RobotClient? _robot;
  List<ResourceName>? _resources;
  String? _error;

  VideoPlayerController? _videoController;
  HttpServer? _videoServer;
  _ProgressiveVideoBuffer? _buffer;
  StreamSubscription<List<int>>? _vsStreamSub;
  bool _videoInitStarted = false;
  int _bufferedBytes = 0;
  String _status = 'Idle';

  bool _initInProgress = false;
  bool _finalAttemptDone = false;

  @override
  void initState() {
    super.initState();
    connectToViam();
  }

  Future<void> connectToViam() async {
    try {
      ensureVideostoreRegistered();
      const host = 'framework-1-main.wcfb1lr0dn.viam.cloud';
      const apiKeyID = '093a5315-d9de-4689-9d2b-e9b95dd9ab84';        // replace securely
      const apiKey = 'b463sf1xqqaeiwpjnhtyzgxu1qi7rgu9';      // replace securely
      final robot = await RobotClient.atAddress(
        host,
        RobotClientOptions.withApiKey(apiKeyID, apiKey),
      );
      final names = await robot.resourceNames;
      setState(() {
        _robot = robot;
        _resources = names;
      });
      print(names);
      final rn = VideoStore.subtype.getResourceName('vs-1'); 
      if (!names.contains(rn)) {
        setState(() {
          _status = 'VideoStore resource vs-1 not found. Available: $names';
          _robot = robot;
        });
        print('VideoStore resource vs-1 not found. Available: $names');
        return;
      }
      final vs = VideoStore.fromRobot(robot, 'vs-1');
      await _runFetches(vs);
    } catch (e) {
      setState(() => _error = e.toString());
      print('Viam connect error: $e');
    }
  }

  String _fmtYmdHms(DateTime d) {
    final y = d.year.toString().padLeft(4, '0');
    final m = d.month.toString().padLeft(2, '0');
    final day = d.day.toString().padLeft(2, '0');
    final hh = d.hour.toString().padLeft(2, '0');
    final mm = d.minute.toString().padLeft(2, '0');
    final ss = d.second.toString().padLeft(2, '0');
    return '$y-$m-$day\_${hh}-$mm-${ss}Z';
 }

  Future<void> _runFetches(VideoStore vs) async {
    // wait for 5 seconds to ensure video data is available
    await Future.delayed(const Duration(seconds: 5));
    final now = DateTime.now().toUtc();
    final to = _fmtYmdHms(now.subtract(const Duration(seconds: 40)));
    final from = _fmtYmdHms(now.subtract(const Duration(seconds: 50)));
    print('fetching between $from and $to');
    await _startLocalServerIfNeeded();
    _buffer = _ProgressiveVideoBuffer();
    _bufferedBytes = 0;
    _videoInitStarted = false;
    setState(() => _status = 'Streaming (progressive)...');
    _vsStreamSub?.cancel();
    final stream = vs.fetchStream(from, to);
    _vsStreamSub = stream.listen(
      (chunk) {
        _buffer?.add(chunk);
        _bufferedBytes += chunk.length;
        // Heuristic: start player after first 32KB (likely contains MP4 header + moov)
        if (!_videoInitStarted && _bufferedBytes > 32 * 1024) {
          _videoInitStarted = true;
          _initVideoPlayer();
        }
        if (mounted) {
          setState(() => _status = 'Streaming... $_bufferedBytes bytes');
        }
      },
      onError: (e) {
        _buffer?.close();
        setState(() {
          _error = 'stream error: $e';
          _status = 'Error';
        });
      },
      onDone: () {
        _buffer?.close();
        setState(() => _status = 'Stream done ($_bufferedBytes bytes)');
      },
      cancelOnError: true,
    );
  }

  Future<void> _startLocalServerIfNeeded() async {
    if (_videoServer != null) return;
    _videoServer = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    print('Local video server on port ${_videoServer!.port}');
    _serveRequests();
  }

  void _serveRequests() {
    _videoServer!.listen((req) async {
      if (req.uri.path != '/video.mp4' || _buffer == null) {
        req.response.statusCode = 404;
        await req.response.close();
        return;
      }
      req.response.headers.set(HttpHeaders.contentTypeHeader, 'video/mp4');
      await for (final chunk in _buffer!.stream()) {
        req.response.add(chunk);
        await req.response.flush();
      }
      await req.response.close();
    });
  }

  Future<void> _initVideoPlayer() async {
    if (_videoServer == null) return;
    final url = 'http://127.0.0.1:${_videoServer!.port}/video.mp4';
    print('Initializing VideoPlayer with $url');
    final old = _videoController;
    _videoController = VideoPlayerController.networkUrl(Uri.parse(url));
    try {
      await _videoController!.initialize();
      _videoController!.setLooping(true);
      await _videoController!.play();
      setState(() {});
    } catch (e) {
      setState(() => _error = 'Video init failed: $e');
    }
    await old?.dispose();
  }

  @override
  void dispose() {
    _vsStreamSub?.cancel();
    _videoController?.dispose();
    _videoServer?.close(force: true);
    _robot?.close();
    super.dispose();
  }

  void _incrementCounter() {
    setState(() {
      // This call to setState tells the Flutter framework that something has
      // changed in this State, which causes it to rerun the build method below
      // so that the display can reflect the updated values. If we changed
      // _counter without calling setState(), then the build method would not be
      // called again, and so nothing would appear to happen.
      _counter++;
    });
  }

  @override
  Widget build(BuildContext context) {
    // This method is rerun every time setState is called, for instance as done
    // by the _incrementCounter method above.
    //
    // The Flutter framework has been optimized to make rerunning build methods
    // fast, so that you can just rebuild anything that needs updating rather
    // than having to individually change instances of widgets.
    return Scaffold(
      appBar: AppBar(
        // TRY THIS: Try changing the color here to a specific color (to
        // Colors.amber, perhaps?) and trigger a hot reload to see the AppBar
        // change color while the other colors stay the same.
        backgroundColor: Theme.of(context).colorScheme.inversePrimary,
        // Here we take the value from the MyHomePage object that was created by
        // the App.build method, and use it to set our appbar title.
        title: Text(widget.title),
      ),
      body: Center(
        // Center is a layout widget. It takes a single child and positions it
        // in the middle of the parent.
        child: Column(
          // Column is also a layout widget. It takes a list of children and
          // arranges them vertically. By default, it sizes itself to fit its
          // children horizontally, and tries to be as tall as its parent.
          //
          // Column has various properties to control how it sizes itself and
          // how it positions its children. Here we use mainAxisAlignment to
          // center the children vertically; the main axis here is the vertical
          // axis because Columns are vertical (the cross axis would be
          // horizontal).
          //
          // TRY THIS: Invoke "debug painting" (choose the "Toggle Debug Paint"
          // action in the IDE, or press "p" in the console), to see the
          // wireframe for each widget.
          mainAxisAlignment: MainAxisAlignment.center,
          children: <Widget>[
            const SizedBox(height: 16),
            Text('Status: $_status'),
            if (_error != null)
              Text(_error!, style: const TextStyle(color: Colors.red)),
            const SizedBox(height: 16),
            if (_videoController != null && _videoController!.value.isInitialized)
              AspectRatio(
                aspectRatio: _videoController!.value.aspectRatio == 0
                    ? 16 / 9
                    : _videoController!.value.aspectRatio,
                child: VideoPlayer(_videoController!),
              )
            else
              const Text('Waiting for video data...'),
          ],
        ),
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: _incrementCounter,
        tooltip: 'Increment',
        child: const Icon(Icons.add),
      ), // This trailing comma makes auto-formatting nicer for build methods.
    );
  }
}

class _ProgressiveVideoBuffer {
  final _chunks = <List<int>>[];
  bool _closed = false;
  final _waiters = <Completer<void>>[];

  void add(List<int> c) {
    _chunks.add(c);
    for (final w in _waiters) {
      if (!w.isCompleted) w.complete();
    }
    _waiters.clear();
  }

  void close() {
    _closed = true;
    for (final w in _waiters) {
      if (!w.isCompleted) w.complete();
    }
    _waiters.clear();
  }

  Stream<List<int>> stream() async* {
    var index = 0;
    while (true) {
      while (index < _chunks.length) {
        yield _chunks[index++];
      }
      if (_closed) break;
      final waiter = Completer<void>();
      _waiters.add(waiter);
      await waiter.future;
    }
  }
}
