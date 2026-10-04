import 'package:flutter/material.dart';

class Home extends StatelessWidget {
  const Home({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
        padding: const EdgeInsets.all(20),
        child: Column(
            // Sections
            children: <Widget>[
              // Rating section
              Container(
                  padding: const EdgeInsets.fromLTRB(5, 20, 5, 20),
                  child: const Row(
                    children: [
                      // Card 1
                      Expanded(
                        child: Column(
                          children: [
                            Card(
                              elevation: 2,
                              child: Column(
                                children: [
                                  Padding(
                                      padding:
                                          EdgeInsets.fromLTRB(20, 20, 20, 40),
                                      child: Align(
                                          alignment: Alignment.topLeft,
                                          child: Text('Your current MMR',
                                              style: TextStyle(fontSize: 12)))),
                                  Padding(
                                      padding:
                                          EdgeInsets.fromLTRB(20, 40, 20, 40),
                                      child: Align(
                                          alignment: Alignment.center,
                                          child: Text('1532',
                                              style: TextStyle(fontSize: 32)))),
                                ],
                              ),
                            ),
                          ],
                        ),
                      ),
                      // Card 2
                      Expanded(
                        child: Column(
                          children: [
                            Card(
                              elevation: 2,
                              child: Column(
                                children: [
                                  Padding(
                                      padding:
                                          EdgeInsets.fromLTRB(20, 20, 20, 40),
                                      child: Align(
                                          alignment: Alignment.topLeft,
                                          child: Text('Your current MMR',
                                              style: TextStyle(fontSize: 12)))),
                                  Padding(
                                      padding:
                                          EdgeInsets.fromLTRB(20, 40, 20, 40),
                                      child: Align(
                                          alignment: Alignment.center,
                                          child: Text('1532',
                                              style: TextStyle(fontSize: 32)))),
                                ],
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  )),
            ]));
  }
}
