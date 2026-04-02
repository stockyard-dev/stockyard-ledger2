package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashHTML))
}

const dashHTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Ledger II</title>
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#c45d2c;--rl:#e8753a;--leather:#a0845c;--ll:#c4a87a;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c44040;--mono:'JetBrains Mono',Consolas,monospace;--serif:'Libre Baskerville',Georgia,serif}
*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--mono);font-size:13px;line-height:1.6}
.hdr{padding:.6rem 1.2rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}
.hdr h1{font-family:var(--serif);font-size:1rem}.hdr h1 span{color:var(--rl)}
.main{max-width:800px;margin:0 auto;padding:1rem 1.2rem}
.btn{font-family:var(--mono);font-size:.68rem;padding:.3rem .6rem;border:1px solid;cursor:pointer;background:transparent;transition:.15s;white-space:nowrap}
.btn-p{border-color:var(--rust);color:var(--rl)}.btn-p:hover{background:var(--rust);color:var(--cream)}
.btn-d{border-color:var(--bg3);color:var(--cm)}.btn-d:hover{border-color:var(--red);color:var(--red)}
.overview{display:flex;gap:1.5rem;margin-bottom:1.2rem;font-size:.7rem;color:var(--leather);flex-wrap:wrap}
.overview .stat b{display:block;font-size:1.4rem;color:var(--cream)}
.tabs{display:flex;gap:0;margin-bottom:1rem;border-bottom:1px solid var(--bg3)}
.tab{padding:.4rem 1rem;cursor:pointer;font-size:.75rem;color:var(--cm);border-bottom:2px solid transparent;transition:.15s}
.tab:hover{color:var(--cream)}.tab.active{color:var(--rl);border-bottom-color:var(--rl)}
.txn-row{display:flex;align-items:center;gap:.5rem;padding:.35rem .5rem;border-bottom:1px solid var(--bg3);font-size:.72rem}
.txn-date{color:var(--cm);width:65px;flex-shrink:0}.txn-payee{flex:1;font-weight:600}.txn-cat{font-size:.6rem;padding:.05rem .25rem;background:var(--bg3);color:var(--ll);border-radius:2px}
.txn-amt{font-weight:600;width:80px;text-align:right}
.income{color:var(--green)}.expense{color:var(--red)}
.acct-row{display:flex;align-items:center;gap:.6rem;padding:.5rem;background:var(--bg2);border:1px solid var(--bg3);margin-bottom:.3rem}
.acct-name{flex:1;font-weight:600}.acct-type{font-size:.6rem;padding:.05rem .25rem;background:var(--bg3);color:var(--ll);border-radius:2px}
.budget-row{background:var(--bg2);border:1px solid var(--bg3);padding:.5rem;margin-bottom:.3rem}
.budget-top{display:flex;justify-content:space-between;font-size:.75rem;margin-bottom:.3rem}
.budget-bar{height:5px;background:var(--bg3);border-radius:3px;overflow:hidden}
.budget-fill{height:100%;transition:width .3s}
.bf-ok{background:var(--green)}.bf-warn{background:var(--gold)}.bf-over{background:var(--red)}
.empty{text-align:center;padding:2rem;color:var(--cm);font-style:italic;font-family:var(--serif)}
.modal-bg{position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,.65);display:flex;align-items:center;justify-content:center;z-index:100}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:95%;max-width:500px;max-height:90vh;overflow-y:auto}
.modal h2{font-family:var(--serif);font-size:.9rem;margin-bottom:1rem}
label.fl{display:block;font-size:.65rem;color:var(--leather);text-transform:uppercase;letter-spacing:1px;margin-bottom:.2rem;margin-top:.5rem}
input[type=text],input[type=number],input[type=date],select{background:var(--bg);border:1px solid var(--bg3);color:var(--cream);padding:.35rem .5rem;font-family:var(--mono);font-size:.78rem;width:100%;outline:none}
.form-row{display:flex;gap:.5rem}.form-row>*{flex:1}
</style>
<link href="https://fonts.googleapis.com/css2?family=Libre+Baskerville:ital@0;1&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
</head><body>
<div class="hdr"><h1><span>Ledger II</span></h1><div style="display:flex;gap:.3rem"><button class="btn btn-p" onclick="showNewTxn()">+ Transaction</button><button class="btn btn-p" onclick="showNewAcct()">+ Account</button></div></div>
<div class="main">
<div class="overview" id="overview"></div>
<div class="tabs">
<div class="tab active" data-tab="transactions" onclick="switchTab('transactions')">Transactions</div>
<div class="tab" data-tab="accounts" onclick="switchTab('accounts')">Accounts</div>
<div class="tab" data-tab="budgets" onclick="switchTab('budgets')">Budgets</div>
</div>
<div id="pane-transactions"><div id="txnList"></div></div>
<div id="pane-accounts" style="display:none"><div id="acctList"></div></div>
<div id="pane-budgets" style="display:none"><div style="display:flex;justify-content:space-between;margin-bottom:.5rem"><span style="font-size:.7rem;color:var(--leather)">Monthly budget envelopes</span><button class="btn btn-p" onclick="showNewBudget()">+ Budget</button></div><div id="budgetList"></div></div>
</div>
<div id="modal"></div>
<script>
let accounts=[],transactions=[];
async function api(u,o){return(await fetch(u,o)).json()}
function esc(s){return String(s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')}
function fmt(n){return(n>=0?'':'-')+'$'+Math.abs(n).toFixed(2)}

async function init(){
  const[ad,td,sd,sm]=await Promise.all([api('/api/accounts'),api('/api/transactions'),api('/api/stats'),api('/api/summary')]);
  accounts=ad.accounts||[];transactions=td.transactions||[];
  const nw=sd.net_worth;
  document.getElementById('overview').innerHTML=
    '<div class="stat"><b style="color:var(--green)">'+fmt(sm.income)+'</b>Income</div>'+
    '<div class="stat"><b style="color:var(--red)">'+fmt(sm.expenses)+'</b>Expenses</div>'+
    '<div class="stat"><b style="color:'+(sm.net>=0?'var(--green)':'var(--red)')+'">'+fmt(sm.net)+'</b>This Month</div>'+
    '<div class="stat"><b>'+fmt(nw)+'</b>Net Worth</div>';
  renderTxns();renderAccounts()
}
function renderTxns(){
  document.getElementById('txnList').innerHTML=transactions.length?transactions.map(t=>
    '<div class="txn-row"><span class="txn-date">'+t.date+'</span><span class="txn-payee">'+esc(t.payee||'—')+'</span>'+
    (t.category?'<span class="txn-cat">'+esc(t.category)+'</span>':'')+
    '<span style="font-size:.6rem;color:var(--cm)">'+esc(t.account_name)+'</span>'+
    '<span class="txn-amt '+(t.amount>=0?'income':'expense')+'">'+fmt(t.amount)+'</span>'+
    '<span style="cursor:pointer;font-size:.55rem;color:var(--cm)" onclick="delTxn(\''+t.id+'\')">x</span></div>'
  ).join(''):'<div class="empty">No transactions yet.</div>'
}
function renderAccounts(){
  document.getElementById('acctList').innerHTML=accounts.length?accounts.map(a=>
    '<div class="acct-row"><span class="acct-type">'+esc(a.type)+'</span><span class="acct-name">'+esc(a.name)+'</span>'+
    '<span style="font-weight:600;color:'+(a.balance>=0?'var(--green)':'var(--red)')+'">'+fmt(a.balance)+'</span>'+
    '<span style="cursor:pointer;font-size:.55rem;color:var(--cm)" onclick="delAcct(\''+a.id+'\')">del</span></div>'
  ).join(''):'<div class="empty">No accounts yet.</div>'
}
async function loadBudgets(){
  const d=await api('/api/budgets');const budgets=d.budgets||[];
  document.getElementById('budgetList').innerHTML=budgets.length?budgets.map(b=>{
    const pct=b.amount?Math.min(100,Math.round(b.spent/b.amount*100)):0;
    const cls=pct>=100?'bf-over':pct>=80?'bf-warn':'bf-ok';
    return'<div class="budget-row"><div class="budget-top"><span style="font-weight:600">'+esc(b.category)+'</span><span>'+fmt(b.spent)+' / '+fmt(b.amount)+'</span></div>'+
      '<div class="budget-bar"><div class="budget-fill '+cls+'" style="width:'+pct+'%"></div></div>'+
      '<div style="font-size:.6rem;color:var(--cm);margin-top:.2rem">'+fmt(b.remaining)+' remaining</div></div>'
  }).join(''):'<div class="empty">No budgets set.</div>'
}
function switchTab(t){
  document.querySelectorAll('.tab').forEach(el=>el.classList.toggle('active',el.dataset.tab===t));
  document.getElementById('pane-transactions').style.display=t==='transactions'?'':'none';
  document.getElementById('pane-accounts').style.display=t==='accounts'?'':'none';
  document.getElementById('pane-budgets').style.display=t==='budgets'?'':'none';
  if(t==='budgets')loadBudgets()
}
function showNewTxn(){
  const opts=accounts.map(a=>'<option value="'+a.id+'">'+esc(a.name)+'</option>').join('');
  document.getElementById('modal').innerHTML='<div class="modal-bg" onclick="if(event.target===this)closeModal()"><div class="modal"><h2>New Transaction</h2><label class="fl">Account</label><select id="nt-acct">'+opts+'</select><label class="fl">Date</label><input type="date" id="nt-date" value="'+new Date().toISOString().split('T')[0]+'"><div class="form-row"><div><label class="fl">Payee</label><input type="text" id="nt-payee" placeholder="Grocery Store"></div><div><label class="fl">Category</label><input type="text" id="nt-cat" placeholder="food"></div></div><label class="fl">Amount (negative for expenses)</label><input type="number" id="nt-amt" step="0.01" placeholder="-42.50"><label class="fl">Note</label><input type="text" id="nt-note"><div style="display:flex;gap:.5rem;margin-top:1rem"><button class="btn btn-p" onclick="saveTxn()">Add</button><button class="btn btn-d" onclick="closeModal()">Cancel</button></div></div></div>'
}
async function saveTxn(){const b={account_id:document.getElementById('nt-acct').value,date:document.getElementById('nt-date').value,payee:document.getElementById('nt-payee').value,category:document.getElementById('nt-cat').value,amount:parseFloat(document.getElementById('nt-amt').value)||0,note:document.getElementById('nt-note').value};if(!b.amount){alert('Amount required');return};await api('/api/transactions',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)});closeModal();init()}
function showNewAcct(){document.getElementById('modal').innerHTML='<div class="modal-bg" onclick="if(event.target===this)closeModal()"><div class="modal"><h2>New Account</h2><label class="fl">Name</label><input type="text" id="na-name"><label class="fl">Type</label><select id="na-type"><option>checking</option><option>savings</option><option>credit</option><option>cash</option><option>investment</option></select><div style="display:flex;gap:.5rem;margin-top:1rem"><button class="btn btn-p" onclick="saveAcct()">Create</button><button class="btn btn-d" onclick="closeModal()">Cancel</button></div></div></div>'}
async function saveAcct(){const b={name:document.getElementById('na-name').value,type:document.getElementById('na-type').value};if(!b.name){alert('Name required');return};await api('/api/accounts',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)});closeModal();init()}
function showNewBudget(){document.getElementById('modal').innerHTML='<div class="modal-bg" onclick="if(event.target===this)closeModal()"><div class="modal"><h2>Set Budget</h2><label class="fl">Category</label><input type="text" id="nb-cat" placeholder="food"><label class="fl">Monthly Amount</label><input type="number" id="nb-amt" step="0.01" placeholder="500"><div style="display:flex;gap:.5rem;margin-top:1rem"><button class="btn btn-p" onclick="saveBudget()">Set</button><button class="btn btn-d" onclick="closeModal()">Cancel</button></div></div></div>'}
async function saveBudget(){const b={category:document.getElementById('nb-cat').value,month:new Date().toISOString().substring(0,7),amount:parseFloat(document.getElementById('nb-amt').value)||0};if(!b.category||!b.amount){alert('Category and amount required');return};await api('/api/budgets',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)});closeModal();loadBudgets()}
async function delTxn(id){await api('/api/transactions/'+id,{method:'DELETE'});init()}
async function delAcct(id){if(!confirm('Delete account and all transactions?'))return;await api('/api/accounts/'+id,{method:'DELETE'});init()}
function closeModal(){document.getElementById('modal').innerHTML=''}
init()
</script></body></html>`
