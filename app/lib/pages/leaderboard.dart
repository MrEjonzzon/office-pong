import 'package:flutter/material.dart';
import 'package:office_pong/pages/profile.dart';

import 'package:office_pong/models/player.dart';

class Leaderboard extends StatefulWidget {
  const Leaderboard({Key? key}) : super(key: key);

  @override
  State<Leaderboard> createState() => _LeaderboardState();
}

class _LeaderboardState extends State<Leaderboard> {
  late Future<List<Player>> players;

  @override
  void initState() {
    super.initState();
    players = fetchPlayers();
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<Player>>(
      future: players,
      builder: (context, snapshot) {
        if (snapshot.hasData) {
          return SingleChildScrollView(
            child: ListView.builder(
                physics: const NeverScrollableScrollPhysics(),
                shrinkWrap: true,
                itemCount: snapshot.data!.length,
                itemBuilder: (BuildContext context, int index) {
                  return ListTile(
                    leading: const CircleAvatar(
                      backgroundImage: AssetImage(
                          'assets/politecat.webp'), // TODO: Real avatar
                    ),
                    title: Text(
                      snapshot.data![index].name,
                      style: const TextStyle(color: Colors.black, fontSize: 18),
                    ),
                    subtitle: Text(
                      snapshot.data![index].team,
                      style: const TextStyle(color: Colors.black, fontSize: 16),
                    ),
                    trailing: Text(
                      snapshot.data![index].mmr.toString(),
                      style: const TextStyle(color: Colors.black, fontSize: 20),
                    ),
                    onTap: () {
                      Navigator.push(
                        context,
                        MaterialPageRoute(
                          builder: (context) => Scaffold(
                            appBar: AppBar(
                              backgroundColor: Colors.transparent,
                              toolbarHeight: 110,
                              title: const Text(
                                'Profile',
                                style: TextStyle(
                                  fontSize: 32,
                                  fontWeight: FontWeight.w200,
                                ),
                              ),
                            ),
                            body: Column(
                              children: <Widget>[
                                Profile(player: snapshot.data![index]),
                                const Spacer(),
                                SafeArea(
                                  child: ElevatedButton.icon(
                                    icon: const Icon(Icons
                                        .flag), // Replace with your desired icon
                                    label: const Text('Challenge'),
                                    style: ElevatedButton.styleFrom(
                                      shape: RoundedRectangleBorder(
                                        borderRadius:
                                            BorderRadius.circular(32.0),
                                      ),
                                      minimumSize: Size(
                                        MediaQuery.of(context).size.width *
                                            8 /
                                            12,
                                        60,
                                      ),
                                    ),
                                    onPressed: () {
                                      // Add functionality for challenging player
                                    },
                                  ),
                                ),
                              ],
                            ), // Assuming Profile takes a player as a parameter
                          ),
                        ),
                      );
                    },
                  );
                }),
          );
        } else if (snapshot.hasError) {
          return Center(child: Text('${snapshot.error}'));
        }

        // Loader
        return const Center(
          child: CircularProgressIndicator(),
        );
      },
    );
  }
}
