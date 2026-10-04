import 'dart:convert';
import 'package:http/http.dart' as http;

Future<List<Player>> fetchPlayers() async {
  final response =
      await http.get(Uri.parse('http://localhost:8080/api/leaderboard'));

  if (response.statusCode == 200) {
    List jsonResponse = json.decode(response.body);
    return jsonResponse.map((item) => Player.fromJson(item)).toList();
  } else {
    throw Exception('Failed to load players from API');
  }
}

class Player {
  final String email;
  final String name;
  final int mmr;
  final String team;

  Player(
      {required this.email,
      required this.name,
      this.mmr = 0,
      this.team = 'Office'});

  factory Player.fromJson(Map<String, dynamic> json) {
    return Player(
      email: json['email'],
      name: json['name'],
      mmr: json['mmr'],
      team: json['team'],
    );
  }
}
