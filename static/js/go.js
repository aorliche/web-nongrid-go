import {$, $$} from './util.js';
import {noFillFn, Board} from './board.js';
import {Point, Edge, Polygon, randomEdgePoint} from './primitives.js';

function initBoard(board) {
    board.loop([(a,b) => board.fill(a,6,b)]);
    board.loop([(a,b) => board.fill(a,3,b)]);
    board.loop([(a,b) => board.fill(a,4,b)]);
    board.loop([(a,b) => board.fill(a,3,b)]);
    board.loop([(a,b) => board.fill(a,4,b)]);
    board.loop([(a,b) => board.fill(a,3,b)]);
    board.loop([noFillFn, (a,b) => board.fill(a,4,b)]);
    board.loop([(a,b) => board.placeOne(a,3,b)]);
    board.loop([(a,b) => board.fill(a,6,b)]);
    board.loop([(a,b) => board.fill(a,3,b)]);
    board.loop([(a,b) => board.fill(a,3,b)]);
    board.initNeighbors();
    board.repaint();
}

function setupListeners(game) {
    game.conn.onmessage = e => {
        const json = JSON.parse(e.data);
        if (json.Action == "New") {
            game.id = json.Key;
            return;
        }
        if (json.Action == "Join" || json.Action == "Move") {
            const pts = JSON.parse(json.Payload);
            game.board.history.push(JSON.stringify(pts));
            game.board.loadPoints(pts);
            game.board.repaint();
            game.board.player = json.Player;
            return;
        }
    }
}   

window.addEventListener('load', () => {
    const canvas = $('#canvas');

    // Separate connection in game object for game
    // This connection only for listing games
    const conn = new WebSocket(`ws://${location.host}/list`);

    let player = null;
    let game = null;

    $('#new').addEventListener('click', () => {
        game = {board: new Board(canvas), player: 'black'};
        initBoard(game.board); 
        const pts = game.board.savePoints();
        game.conn = new WebSocket(`ws://${location.host}/ws`);
        game.conn.onopen = () => {
            game.conn.send(JSON.stringify({Action: 'New', Payload: JSON.stringify(pts)}));
        };
        setupListeners(game);
    });

    $('#join').addEventListener('click', () => {
        game = {board: new Board(canvas), player: 'white'};
        initBoard(game.board); 
        const sel = $('select[name="games-list"]');    
        const id = sel.options[sel.selectedIndex].value;
        game.conn = new WebSocket(`ws://${location.host}/ws`);
        game.id = parseInt(id);
        game.conn.onopen = () => {
            game.conn.send(JSON.stringify({Action: 'Join', Key: game.id}));
        };
        setupListeners(game);
    });
    
    $('#canvas').addEventListener('mousemove', (e) => {
        if (!game || !game.board || game.player != game.board.player) return;
        game.board.hover(e.offsetX, e.offsetY);
        game.board.repaint();
    });

    $('#canvas').addEventListener('click', (e) => {
        if (!game || !game.board || game.player != game.board.player) return;
        const res = game.board.click(e.offsetX, e.offsetY);
        if (!res) return;
        game.board.repaint();
        game.conn.send(JSON.stringify({Action: 'Move', Key: game.id, Payload: JSON.stringify(game.board.savePoints())}));
    });

    // This conn only used for listing games
    conn.onmessage = e => {
        const json = JSON.parse(e.data);

        // List of integer game ids
        json.sort((a,b) => a-b);

        const select = $('select[name="games-list"]');
        const toAdd = [];
        const games = [...select.options].map(opt => parseInt(opt.value));
        if (game && games.includes(game.id)) {
            for (let i=0; i<select.options.length; i++) {
                const opt = select.options[i];
                if (parseInt(opt.value) == game.id) {
                    select.remove(i);
                    break;
                }
            }
        } 
        for (let i=0; i<select.options.length; i++) {
            const opt = select.options[i];
            if (!json.includes(parseInt(opt.value))) {
                select.remove(i--);
            }
        }
        json.forEach(key => {
            if (!games.includes(key) && !(game && game.id == key)) {
                const opt = document.createElement('option');
                opt.value = key;
                opt.innerHTML = `Game ${key}`;
                select.appendChild(opt);
            }
        });
    }

    setInterval(e => {
        if (!conn.readyState == 1) return;
        conn.send(JSON.stringify({Action: 'List'}));
    }, 1000);
});
