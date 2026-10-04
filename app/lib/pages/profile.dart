import 'package:flutter/material.dart';
import 'package:office_pong/models/player.dart';

class Profile extends StatelessWidget {
  final Player player;

  const Profile({Key? key, required this.player}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: <Widget>[
          Text(
            '${player.mmr} MMR',
            style: const TextStyle(
              fontSize: 40,
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 10),
          const CircleAvatar(
              radius: 50, backgroundImage: AssetImage('assets/politecat.webp')),
          const SizedBox(height: 20),
          Text(
            player.name,
            style: const TextStyle(
              fontSize: 24,
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 10),
          Text(
            player.team,
            style: const TextStyle(
              fontSize: 16,
              color: Colors.grey,
            ),
          ),
          const SizedBox(height: 20),
          // ElevatedButton(
          //   onPressed: () {
          //     // Add functionality for editing profile
          //   },
        ],
      ),
    );
  }
}
