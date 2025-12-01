import React, { useState, useEffect, useRef } from 'react';

// --- STYLES 
const styles = {
  container: { fontFamily: 'Arial, sans-serif', textAlign: 'center', backgroundColor: '#282c34', minHeight: '100vh', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: 'white' },
  input: { padding: '10px', fontSize: '16px', borderRadius: '5px', border: 'none', marginRight: '10px' },
  button: { padding: '10px 20px', fontSize: '16px', borderRadius: '5px', border: 'none', cursor: 'pointer', backgroundColor: '#61dafb', color: '#282c34', fontWeight: 'bold' },
  board: { display: 'grid', gridTemplateColumns: 'repeat(7, 50px)', gap: '5px', backgroundColor: '#0055aa', padding: '10px', borderRadius: '10px', margin: '20px auto' },
  cell: { width: '50px', height: '50px', backgroundColor: 'white', borderRadius: '50%', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' },
  p1: { backgroundColor: '#ff4136', width: '100%', height: '100%', borderRadius: '50%' },
  p2: { backgroundColor: '#ffdc00', width: '100%', height: '100%', borderRadius: '50%' },
  status: { fontSize: '24px', marginBottom: '20px' },
  log: { marginTop: '20px', fontSize: '12px', color: '#aaa', maxHeight: '100px', overflowY: 'auto' }
};

export default function App() {
  // Game State
  const [socket, setSocket] = useState(null);
  const [view, setView] = useState('login'); // login, matching, game, gameover
  const [username, setUsername] = useState('');
  const [gameInfo, setGameInfo] = useState({ opponent: '', symbol: 0, isTurn: false });
  const [board, setBoard] = useState(Array(6).fill(null).map(() => Array(7).fill(0)));
  const [winner, setWinner] = useState(null);
  
  // Initialize WebSocket
  useEffect(() => {
    const ws = new WebSocket('ws://localhost:8080/ws');
    
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

      default:
        break;
    }
  };

  const joinGame = () => {
    if (!username || !socket) return;
    socket.send(JSON.stringify({ type: 'JOIN', payload: { username } }));
    setView('matching');
  };

  const makeMove = (colIndex) => {
    if (!gameInfo.isTurn || view !== 'game') return;
    // Optimistic UI update (optional, but makes it feel faster)
    // For now, we wait for server response to be safe
    socket.send(JSON.stringify({ type: 'MOVE', payload: { column: colIndex } }));
  };

  // --- RENDERERS ---

  if (view === 'login') {
    return (
      <div style={styles.container}>
        <h1>Connect 4</h1>
        <div>
          <input 
            style={styles.input} 
            placeholder="Enter Username" 
            value={username} 
            onChange={e => setUsername(e.target.value)} 
          />
          <button style={styles.button} onClick={joinGame}>Find Match</button>
        </div>
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
        {/* Render 7 Columns (Clickable) */}
        {Array(7).fill(0).map((_, colIndex) => (
          <div key={colIndex} onClick={() => makeMove(colIndex)} style={{display: 'flex', flexDirection: 'column'}}>
            {/* Render 6 Rows per column */}
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
         <button style={styles.button} onClick={() => window.location.reload()}>Play Again</button>
      )}
      
      <div style={styles.log}>
        You are Player {gameInfo.symbol} vs {gameInfo.opponent}
      </div>
    </div>
  );
}