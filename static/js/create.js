
import {$, $$} from './util.js';
import {noFillFn, Board} from './board.js';

window.addEventListener('load', function(){
    const fillplan = [];
    let boardplan = [];
    function regenFillPlan() {
        $('#fillplan').innerHTML = '';
        for (let i=0; i<fillplan.length; i++) {
            const li = document.createElement('li');
            const button = document.createElement('button');
            const txt = document.createTextNode(fillplan[i].txt);
            button.innerText = 'Delete';
            li.appendChild(txt);
            li.appendChild(button);
            $('#fillplan').appendChild(li);
            button.addEventListener('click', e => {
                fillplan.splice(i, 1);
                regenFillPlan(); 
            });
        }
    }
    function regenBoardPlan() {
        $('#boardplan').innerHTML = '';
        for (let i=0; i<boardplan.length; i++) {
            const li = document.createElement('li');
            const button = document.createElement('button');
            const round = [];
            for (let j=0; j<boardplan[i].sav.length; j++) {
                round.push(boardplan[i].sav[j].txt);
            }
            const txt = document.createTextNode(`${boardplan[i].typ}: ${round}`);
            button.innerText = 'Delete';
            li.appendChild(txt);
            li.appendChild(button);
            $('#boardplan').appendChild(li);
            button.addEventListener('click', e => {
                boardplan.splice(i, 1);
                regenBoardPlan(); 
            });
        }

    }
    function addShape(n) {
        switch (n) {
            case 0: fillplan.push({n, txt: 'Skip'}); break;
            case 3: fillplan.push({n, txt: 'Triangles'}); break;
            case 4: fillplan.push({n, txt: 'Squares'}); break;
            case 6: fillplan.push({n, txt: 'Hexagons'}); break;
        }
        regenFillPlan();
    }
    $('#nofill').addEventListener('click', e => addShape(0));
    $('#fill3').addEventListener('click', e => addShape(3));
    $('#fill4').addEventListener('click', e => addShape(4));
    $('#fill6').addEventListener('click', e => addShape(6));
    let board = new Board($('#canvas'));
    function fillOrPlaceBtnCb(typ, fn) {
        const arr = [];
        const sav = [];
        for (let i=0; i<fillplan.length; i++) {
            sav.push(fillplan[i]);
            switch (fillplan[i].n) {
                case 0: arr.push(noFillFn); break;
                case 3: arr.push(fn(3)); break;
                case 4: arr.push(fn(4)); break;
                case 6: arr.push(fn(6)); break;
            }
        }
        board.loop(arr);
        board.repaint();
        // Save for regen
        boardplan.push({typ, sav});
        regenBoardPlan();
    }
    $('#fill').addEventListener('click', e => {
        fillOrPlaceBtnCb('fill', n => {
            return (a, b) => board.fill(a,n,b);
        });
    });
    $('#place').addEventListener('click', e => {
        fillOrPlaceBtnCb('place', n => {
            return (a, b) => board.placeOne(a,n,b);
        });
    });
    function repaintFromBoardPlan() {
        board = new Board($('#canvas'));
        const fn = (typ, n) => {
            if (n == 0) {
                return noFillFn;
            } else if (typ == 'fill') {
                return (a, b) => board.fill(a,n,b);
            } else {
                return (a, b) => board.placeOne(a,n,b);
            }
        }
        boardplan.forEach(round => {
            const arr = [];
            for (let i=0; i<round.sav.length; i++) {
                arr.push(fn(round.typ, round.sav[i].n));
            }
            board.loop(arr);
        });
        board.repaint();
    }
    $('#regen').addEventListener('click', e => {
        repaintFromBoardPlan();
    });
    
    const conn = new WebSocket(`ws://${location.host}/boards`);

    $('#upload').addEventListener('click', e => {
        const name = $('#name').value.trim();
        if (name == '' || boardplan.length == 0) {
            alert('Name is empty or no boardplan');
            return;
        }
        const req = {Action: 'Save', Player: name, Payload: JSON.stringify(boardplan)};
        conn.send(JSON.stringify(req));
    });

    $('#load').addEventListener('click', e => {
        const idx = $('#boards').selectedIndex;
        if (idx == -1) return;
        const req = {Action: 'Load', Player: $('#boards').options[idx].innerText};
        conn.send(JSON.stringify(req));
    });

    conn.onmessage = e => {
        const msg = JSON.parse(e.data);
        if (msg.Action == 'List') {
            JSON.parse(msg.Payload).forEach(name => {
                const existing = $$('#boards option');
                let found = false;
                for (let i=0; i<existing.length; i++) {
                    if (existing[i].innerText == name) {
                        found = true;
                        break;
                    }
                }
                if (!found) {
                    const opt = document.createElement('option');
                    opt.innerText = name;
                    $('#boards').appendChild(opt);
                }
            });
        } else if (msg.Action == 'Load') {
            boardplan = JSON.parse(msg.Payload);
            regenBoardPlan();
            repaintFromBoardPlan();
        } else if (msg.Action == 'Save') {
            alert(msg.Payload);
        }
    }
    
    setInterval(e => {
        if (!conn.readyState == 1) return;
        conn.send(JSON.stringify({Action: 'List'}));
    }, 1000);
});
