package main

var matrix = [][]float32{
	/* ChallengeCmd, QueueCmd, MakeMicrotransactionCmd, TradeCmd, StoreCmd, CatchCmd, RaidCmd */
	/* c,   q,    m,    t,    s,    c     r*/
	{0.05, 0.10, 0.10, 0.20, 0.10, 0.30, 0.15}, // previousMove = Challenge
	{0.15, 0.05, 0.10, 0.15, 0.10, 0.30, 0.15}, // previousMove = Queue
	{0.15, 0.10, 0.05, 0.15, 0.10, 0.30, 0.15}, // previousMove = MakeMicrotransactionCmd
	{0.20, 0.10, 0.10, 0.05, 0.10, 0.30, 0.15}, // previousMove = Trade
	{0.15, 0.10, 0.10, 0.15, 0.05, 0.30, 0.15}, // previousMove = Store
	{0.20, 0.10, 0.10, 0.20, 0.10, 0.15, 0.15}, // previousMove = Catch
	{0.15, 0.10, 0.10, 0.15, 0.15, 0.30, 0.05}, // previousMove = Raid
}
