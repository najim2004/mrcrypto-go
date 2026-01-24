# Trading Bot Execution Flow

This document outlines the step-by-step execution flow of the trading bot `mrcrypto-go`.

## High-Level Overview

1. **Initialization**: Services (Binance, Strategy, AI, Database, Telegram) are initialized.
2. **Scheduler**: A cron job triggers the polling cycle every **1 minute**.
3. **Polling Cycle**:
   - Fetches target symbols from Binance.
   - Distributes symbols to a **Worker Pool** (parallel processing).
   - Workers evaluate each symbol using the **Strategy**.
   - Valid signals are collected.
4. **Filtering**: Signals are filtered by **Cooldown** (4 hours).
5. **Validation**: Remaining signals are validated by **Google Gemini AI**.
6. **Action**: Approved signals (Score ≥ 70) are saved to MongoDB and sent to Telegram.

---

## Detailed Flowchart

```mermaid
graph TD
    Start([Start Loop @ 1min]) --> FetchSymbols[Fetch Symbols (Binance)]
    FetchSymbols --> WorkerPool[Distribute to Worker Pool]
    
    subgraph "Worker Logic (Per Symbol)"
        WorkerPool --> FetchData[Fetch Klines (4h, 1h, 15m, 5m)]
        FetchData --> CheckData{Data Sufficient?}
        CheckData -- No --> StopWorker([Stop / Next Symbol])
        CheckData -- Yes --> CalcInd[Calculate Indicators\n(RSI, ADX, VWAP, MACD)]
        
        CalcInd --> DetectRegime[Detect Market Regime]
        DetectRegime --> CheckChoppy{ADX < 20?}
        CheckChoppy -- Yes (Choppy) --> StopWorker
        CheckChoppy -- No --> CheckPremium{Check PREMIUM Tier}
        
        CheckPremium -- Pass --> CreatePremium[Create PREMIUM Signal]
        CheckPremium -- Fail --> CheckStandard{Check STANDARD Tier}
        
        CheckStandard -- Pass --> CreateStandard[Create STANDARD Signal]
        CheckStandard -- Fail --> StopWorker
        
        CreatePremium --> ReturnSignal([Return Signal])
        CreateStandard --> ReturnSignal
    end
    
    ReturnSignal --> CollectSignals[Collect All Signals]
    CollectSignals --> CountCheck{Signals > 0?}
    CountCheck -- No --> EndCycle([End Cycle])
    
    CountCheck -- Yes --> FilterCooldown[Filter: Check Cooldown]
    
    subgraph "Main Process Logic"
        FilterCooldown --> CooldownCheck{Cooldown Active?\n(Last 4 Hours)}
        CooldownCheck -- Yes --> LogSkip[Log: Skipped (Cooldown)]
        CooldownCheck -- No --> AddToBatch[Add to AI Batch]
        
        LogSkip --> NextSignal
        AddToBatch --> NextSignal{More Signals?}
        NextSignal -- Yes --> FilterCooldown
        NextSignal -- No --> BatchAI[AI Batch Validation]
    end
    
    BatchAI --> AIResults[Process AI Results]
    AIResults --> ScoreCheck{Score >= 70?}
    
    ScoreCheck -- No --> LogLowScore[Log: Score Too Low]
    ScoreCheck -- Yes --> SaveDB[Save to MongoDB]
    
    SaveDB --> SendTelegram[Send Telegram Notification]
    SendTelegram --> EndCycle
    LogLowScore --> EndCycle
```

---

## Decision Logic & Thresholds

### 1. Market Regime Detection
| Condition | Regime | Action |
| :--- | :--- | :--- |
| **ADX (4h) < 20** | `Choppy` | ❌ **STOP** (No trades) |
| Price > EMA50 | `Trending Up` | Potential **LONG** |
| Price < EMA50 | `Trending Down` | Potential **SHORT** |
| Price == EMA50 | `Ranging` | Skip |

### 2. Tier Evaluation (Worker Level)

#### **PREMIUM Tier** (Checked First)
*If ALL conditions are true:*
*   **ADX (1h)** ≥ 25
*   **Volume** ≥ 2.0x Average Volume
*   **Trending Up (LONG)**:
    *   RSI (1h): 50 - 65
    *   RSI (5m): 40 - 70
    *   Order Flow > 0
    *   MACD Histogram > 0
*   **Trending Down (SHORT)**:
    *   RSI (1h): 35 - 50
    *   RSI (5m): 30 - 60
    *   Order Flow < 0
    *   MACD Histogram < 0

#### **STANDARD Tier** (Checked if Premium Fails)
*If ALL conditions are true:*
*   **ADX (1h)** ≥ 20
*   **Volume** ≥ 1.0x Average Volume
*   **Trending Up (LONG)**:
    *   RSI (1h): 40 - 70
    *   RSI (5m): 35 - 75
    *   MACD Histogram > 0
*   **Trending Down (SHORT)**:
    *   RSI (1h): 30 - 60
    *   RSI (5m): 25 - 65
    *   MACD Histogram < 0

### 3. Filters & Validation (Main Process)

*   **Cooldown**: If a signal was generated for this symbol in the last **4 Hours**, it is skipped.
*   **AI Validation**:
    *   **Input**: Technical indicators (RSI, ADX, VWAP, MACD, Volume) for the signal.
    *   **Threshold**: `Score ≥ 70`
    *   **Action**:
        *   Pass: Proceed to Save/Send.
        *   Fail: Log "AI score too low" and discard.

### 4. Risk Management (Signal Creation)

*   **Stop Loss (SL)**: 2% from Entry.
*   **Take Profit (TP)**: 6% from Entry.
*   **Risk/Reward**: 1:3
