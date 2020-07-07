package main

var matrix = [][]float32{
	/* ChallengeCmd, QueueCmd, MakeMicrotransactionCmd, TradeCmd, StoreCmd, CatchCmd, RaidCmd */
	/* c,   q,    m,,  t,    s,    c,	 r*/
	{0.05, 0.05, 0.10, 0.20, 0.20, 0.30, 0.10}, // previousMove = Challenge
	{0.05, 0.05, 0.10, 0.20, 0.20, 0.30, 0.10}, // previousMove = Queue
	{0.05, 0.05, 0.10, 0.20, 0.20, 0.30, 0.10}, // previousMove = MakeMicrotransactionCmd
	{0.05, 0.05, 0.10, 0.20, 0.20, 0.30, 0.10}, // previousMove = Trade
	{0.05, 0.05, 0.10, 0.20, 0.20, 0.30, 0.10}, // previousMove = Store
	{0.05, 0.05, 0.10, 0.20, 0.20, 0.30, 0.10}, // previousMove = Catch
	{0.05, 0.05, 0.10, 0.20, 0.20, 0.30, 0.10}, // previousMove = Raid
}
