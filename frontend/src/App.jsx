import React, { useState, useEffect } from 'react';

// --- CONFIGURATION ---
const API_URL = (import.meta.env.VITE_API_URL || 'http://localhost:8080').replace(/\/$/, '');
const WS_URL = (import.meta.env.VITE_WS_URL || 'ws://localhost:8080').replace(/\/$/, '');

// --- STYLES ---
const styles = {
  container: { fontFamily: 'Arial, sans-serif', textAlign: 'center', backgroundColor: '#282c34', minHeight: '100vh', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: 'white' },
  input: { padding: '10px', fontSize: '16px', borderRadius: '5px', border: 'none', marginRight: '10px' },
  button: { padding: '10px 20px', fontSize: '16px', borderRadius: '5px', border: 'none', cursor: 'pointer', backgroundColor: '#61dafb', color: '#282c34', fontWeight: 'bold', margin: '5px' },
  secondaryButton: { padding: '10px 20px', fontSize: '14px', borderRadius: '5px', border: '1px solid #61dafb', cursor: 'pointer', backgroundColor: 'transparent', color: '#61dafb', margin: '5px' },
  board: { display: 'grid', gridTemplateColumns: 'repeat(7, 50px)', gap: '5px', backgroundColor: '#0055aa', padding: '10px', borderRadius: '10px', margin: '20px auto' },
  cell: { width: '50px', height: '50px', backgroundColor: 'white', borderRadius: '50%', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' },
  p1: { backgroundColor: '#ff4136', width: '100%', height: '100%', borderRadius: '50%' },
  p2: { backgroundColor: '#ffdc00', width: '100%', height: '100%', borderRadius: '50%' },
  status: { fontSize: '24px', marginBottom: '20px' },
  table: { width: '300px', margin: '20px auto', borderCollapse: 'collapse', textAlign: 'left' },
  th: { borderBottom: '1px solid #666', padding: '10px', color: '#61dafb' },
  td: { borderBottom: '1px solid #444', padding: '10px' },
  log: { marginTop: '20px', fontSize: '12px', color: '#aaa' }
};

export default function App() {
  const [socket, setSocket] = useState(null);
  const [view, setView] = useState('login'); 
  const [username, setUsername] = useState('');
  const [gameInfo, setGameInfo] = useState({ opponent: '', symbol: 0, isTurn: false });
  const [board, setBoard] = useState(Array(6).fill(null).map(() => Array(7).fill(0)));
  const [winner, setWinner] = useState(null);
  const [leaderboardData, setLeaderboardData] = useState([]);
  
  useEffect(() => {
    console.log("Connecting to WebSocket:", `${WS_URL}/ws`);
    const ws = new WebSocket(`${WS_URL}/ws`);
    
    ws.onopen = () => console.log("Connected to WS");
    ws.onmessage = (event) => handleMessage(JSON.parse(event.data));
    ws.onclose = () => console.log("Disconnected");

    setSocket(ws);
    return () => ws.close();
  }, []);

  const handleMessage = (msg) => {
    console.log("RX:", msg);
    switch (msg.type) {
      case 'START':
        setGameInfo({ 
          opponent: msg.payload.opponent, 
          symbol: msg.payload.symbol, 
          isTurn: msg.payload.isTurn 
        });
        // Reset board and winner state when a new game starts
        setBoard(Array(6).fill(null).map(() => Array(7).fill(0)));
        setWinner(null);
        setView('game');
        break;
      case 'UPDATE':
        setBoard(msg.payload.board);
        setGameInfo(prev => ({ ...prev, isTurn: msg.payload.isYourTurn }));
        break;
      case 'GAME_OVER':
        setWinner(msg.payload.winner);
        setView('gameover');
        break;
      default: break;
    }
  };

  const joinGame = () => {
    if (!username || !socket) return;
    socket.send(JSON.stringify({ type: 'JOIN', payload: { username } }));
    setView('matching');
  };

  const makeMove = (colIndex) => {
    if (!gameInfo.isTurn || view !== 'game') return;
    socket.send(JSON.stringify({ type: 'MOVE', payload: { column: colIndex } }));
  };

  const fetchLeaderboard = async () => {
    try {
      console.log("Fetching Leaderboard from:", `${API_URL}/leaderboard`);
      const response = await fetch(`${API_URL}/leaderboard`);
      const data = await response.json();
      
      const humanOnlyData = (data || []).filter(entry => entry.username !== 'Bot');
      
      setLeaderboardData(humanOnlyData);
      setView('leaderboard');
    } catch (error) {
      console.error("Failed to fetch leaderboard", error);
      alert("Could not fetch leaderboard. Is the backend running?");
    }
  };

  // --- RENDERERS ---

  if (view === 'login') {
    return (
      <div style={styles.container}>
        <h1>Connect 4</h1>
        <div style={{marginBottom: '20px'}}>
          <input 
            style={styles.input} 
            placeholder="Enter Username" 
            value={username} 
            onChange={e => setUsername(e.target.value)} 
          />
          <button style={styles.button} onClick={joinGame}>Find Match</button>
        </div>
        <button style={styles.secondaryButton} onClick={fetchLeaderboard}>🏆 View Leaderboard</button>
      </div>
    );
  }

  if (view === 'leaderboard') {
    return (
      <div style={styles.container}>
        <h1>🏆 Leaderboard</h1>
        <table style={styles.table}>
          <thead>
            <tr>
              <th style={styles.th}>Rank</th>
              <th style={styles.th}>Player</th>
              <th style={styles.th}>Wins</th>
            </tr>
          </thead>
          <tbody>
            {leaderboardData.length > 0 ? leaderboardData.map((entry, index) => (
              <tr key={index}>
                <td style={styles.td}>#{index + 1}</td>
                <td style={styles.td}>{entry.username}</td>
                <td style={styles.td}>{entry.wins}</td>
              </tr>
            )) : (
              <tr><td colSpan="3" style={styles.td}>No games played yet</td></tr>
            )}
          </tbody>
        </table>
        <button style={styles.secondaryButton} onClick={() => setView('login')}>Back to Home</button>
      </div>
    );
  }

  if (view === 'matching') {
    return (
      <div style={styles.container}>
        <h2>Looking for opponent...</h2>
        <p>If no one joins in 10s, you will play the Bot.</p>
        <div className="loader">⏳</div>
      </div>
    );
  }

  return (
    <div style={styles.container}>
      <div style={styles.status}>
        {view === 'gameover' ? (
          <h2 style={{color: '#00ff00'}}>Winner: {winner}!</h2>
        ) : (
          <h3>
            {gameInfo.isTurn ? "🟢 YOUR TURN" : `🔴 ${gameInfo.opponent}'s Turn`}
          </h3>
        )}
      </div>

      <div style={styles.board}>
        {Array(7).fill(0).map((_, colIndex) => (
          <div key={colIndex} onClick={() => makeMove(colIndex)} style={{display: 'flex', flexDirection: 'column'}}>
            {board.map((row, rowIndex) => {
              const cellValue = board[rowIndex][colIndex];
              return (
                <div key={`${rowIndex}-${colIndex}`} style={styles.cell}>
                  {cellValue === 1 && <div style={styles.p1} />}
                  {cellValue === 2 && <div style={styles.p2} />}
                </div>
              );
            })}
          </div>
        ))}
      </div>

      {view === 'gameover' && (
         <div style={{display: 'flex', gap: '10px'}}>
           <button style={styles.button} onClick={() => window.location.reload()}>Play Again</button>
           <button style={styles.secondaryButton} onClick={fetchLeaderboard}>View Leaderboard</button>
         </div>
      )}
      
      <div style={styles.log}>
        You are Player {gameInfo.symbol} vs {gameInfo.opponent}
      </div>
    </div>
  );
}