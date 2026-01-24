# Trading Bot Execution Flow (Visual)

This flowchart visualizes the complete logic of the `mrcrypto-go` trading bot.

```text
+=================================================================================+
|                                      START                                      |
|                       (Cron Job triggers every 1 Minute)                        |
+=======================================+=========================================+
                                        |
                                        v
                            +-----------------------+
                            | 1. FETCH GLOBAL DATA  |
                            |   Get BTCUSDT (4H)    |
                            +-----------+-----------+
                                        |
                                        v
                            +-----------------------+
                            | 2. DISTRIBUTE TASKS   |
                            |  Send Symbols to Pool |
                            +-----------+-----------+
                                        |
                                        | (Worker execution per symbol)
                                        v
+---------------------------------------------------------------------------------+
|                                 WORKER LOGIC                                    |
+---------------------------------------------------------------------------------+
|                                                                                 |
|  +-----------------------+      +-----------------------+                       |
|  | 3. FETCH KLINES       | ---> | 4. CHECK DATA         | --NO--> [ STOP ]      |
|  | (1D, 4H, 1H, 15M, 5M) |      | Are arrays full?      |                       |
|  +-----------------------+      +-----------+-----------+                       |
|                                             | YES                               |
|                                             v                                   |
|                                 +-----------------------+                       |
|                                 | 5. CALC KEY LEVELS    |                       |
|                                 | • Pivot Points (Daily)|                       |
|                                 | • Fib Levels (4H)     |                       |
|                                 +-----------+-----------+                       |
|                                             |                                   |
|                                             v                                   |
|  +-----------------------+      +-----------------------+                       |
|  | 7. ADVANCED INDICATORS| <--- | 6. CALC INDICATORS    |                       |
|  | • SMC (OB, FVG)       |      | • RSI, ADX, MACD      |                       |
|  | • Volume Profile (POC)|      | • VWAP, Order Flow    |                       |
|  +-----------+-----------+      +-----------------------+                       |
|              |                                                                  |
|              v                                                                  |
|  +-----------------------+      +-----------------------+                       |
|  | 8. MARKET REGIME      | ---> | 9. CHECK CHOPPY       | --YES-> [ STOP ]      |
|  | (Trending/Ranging?)   |      | Is ADX < 15?          |                       |
|  +-----------------------+      +-----------+-----------+                       |
|                                             | NO                                |
|                                             v                                   |
|                                 +-----------------------+                       |
|                                 | 10. DETERMINE DIR     |                       |
|                                 | LONG / SHORT / NONE   | --NONE-> [ STOP ]     |
|                                 +-----------+-----------+                       |
|                                             |                                   |
|                                             v                                   |
|                                 +-----------------------+                       |
|                                 | 11. BTC CHECK         |                       |
|                                 | Align with BTC Trend? |                       |
|                                 +-----------+-----------+                       |
|                                    /                 \                          |
|                             (Aligned)              (Contra)                     |
|                           [ +15 Points ]         [ -20 Penalty ]                |
|                                    \                 /                          |
|                                     v               v                           |
|                                 +-----------------------+                       |
|                                 | 12. SCORING SYSTEM    |                       |
|                                 | • Trend & RSI         |                       |
|                                 | • SMC (OB/FVG) [+15]  |                       |
|                                 | • Vol Profile [+15]   |                       |
|                                 +-----------+-----------+                       |
|                                             |                                   |
|                                             v                                   |
|  +-----------------------+      +-----------------------+                       |
|  | 14. CHECK TIERS       | <--- | 13. SCORE CHECK       | --NO--> [ STOP ]      |
|  | • Premium (Score>=80) |      | Is Score >= 60?       |                       |
|  | • Standard(Score>=60) |      +-----------------------+                       |
|  +-----------+-----------+                                                      |
|              |                                                                  |
|              v                                                                  |
|  +-----------------------+      +-----------------------+                       |
|  | 15. RISK CALCULATION  | ---> | 16. VALIDATE R:R      | --NO--> [ STOP ]      |
|  | • SL (2%), TP (6%)    |      | Is R:R >= 2.0?        |                       |
|  +-----------------------+      +-----------+-----------+                       |
|                                             | YES                               |
|                                             v                                   |
|                                        [ RETURN SIGNAL ]                        |
|                                                                                 |
+=================================================================================+
                                        |
                                        v
                            +-----------------------+
                            | 17. PROCESS SIGNALS   |
                            | Collect valid outputs |
                            +-----------+-----------+
                                        |
                                        v
                            +-----------------------+
                            | 18. AI FILTER (Gemini)|
                            | Validate Logic        |
                            +-----------+-----------+
                                   |          |
                                (Low)       (High)
                                  |           |
                              [ DROP ]    +=================+
                                          | ✅ SAVE & SEND |
                                          | (DB + Telegram) |
                                          +=================+
```

## Flow Description

1.  **Global Data**: The system first checks the general market health by analyzing Bitcoin's 4-hour trend.
2.  **Worker Distribution**: Each coin is processed in parallel to ensure speed.
3.  **Data Fetching**: We fetch 5 different timeframes of data (1D, 4H, 1H, 15m, 5m).
4.  **Indicator Logic**: We calculate standard indicators (RSI, MACD) and advanced ones (Smart Money Concepts, Volume Profile).
5.  **Scoring**:
    *   **BTC Alignment**: We reward following the market leader (+15) and punish fighting it (-20).
    *   **SMC Bonus**: If price is in an Order Block, we add +15 points.
    *   **VP Bonus**: If price is near the Point of Control (Highest Volume), we add +15 points.
6.  **Gatekeeping**:
    *   If Score < 60 -> **Reject**
    *   If Risk:Reward < 1:2 -> **Reject**
    *   If Market is Choppy (ADX < 15) -> **Reject**
7.  **AI Final Check**: Gemini acts as a final filter to remove false positives.
