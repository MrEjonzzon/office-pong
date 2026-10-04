import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:office_pong/pages/home.dart';
import 'package:office_pong/pages/leaderboard.dart';
import 'package:office_pong/pages/profile.dart';

import 'package:flutter/rendering.dart';
import 'package:office_pong/models/player.dart' as player;

void main() {
  debugPaintSizeEnabled = false;
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
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.primaries.first),
        textTheme: GoogleFonts.montserratTextTheme(
          Theme.of(context).textTheme,
        ),
        useMaterial3: true,
      ),
      debugShowCheckedModeBanner: false,
      home: const MyHomePage(title: 'Office Pong'),
    );
  }
}

// State widget
class MyHomePage extends StatefulWidget {
  const MyHomePage({super.key, required this.title});

  final String title;

  @override
  State<MyHomePage> createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  int _selectedIndex = 0;
  final controller = PageController(initialPage: 0);
  final List<String> _titles = ['Home', 'Leaderboard', 'Profile'];

  // On menu item tapped
  void _onItemTapped(int index) {
    setState(() {
      if (kDebugMode) {
        print("index: $index");
      }
      _selectedIndex = index;
      controller.animateToPage(index,
          duration: const Duration(milliseconds: 200), curve: Curves.linear);
    });
  }

  final List<Widget> _pages = <Widget>[
    const Home(),
    const Leaderboard(),
    Profile(
      player: player.Player(
          email: 'player@example.com',
          name: 'Player One',
          mmr: 1500),
    ),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        backgroundColor: Colors.transparent,
        toolbarHeight: 110,
        title: Text(
          _titles[_selectedIndex], // Change this line
          style: const TextStyle(
            fontSize: 32,
            fontWeight: FontWeight.w200, // Add this line
          ),
        ),
      ),
      /* CONTENT */
      body: PageView(
        controller: controller,
        onPageChanged: (value) {
          setState(() {
            _selectedIndex = value;
          });
        },
        children: _pages,
      ),

      /* NAVIGATION */
      bottomNavigationBar: NavigationBar(
        selectedIndex: _selectedIndex,
        labelBehavior: NavigationDestinationLabelBehavior.alwaysHide,
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.home),
            label: 'Home',
            key: Key('home'),
          ),
          NavigationDestination(
            icon: Icon(Icons.leaderboard),
            label: 'Leaderboard',
            key: Key('leaderboard'),
          ),
          NavigationDestination(
            icon: Icon(Icons.person),
            label: 'Profile',
            key: Key('Profile'),
          ),
        ],
        onDestinationSelected: _onItemTapped,
      ),
    );
  }
}
