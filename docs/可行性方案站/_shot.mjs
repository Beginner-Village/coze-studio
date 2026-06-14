import { chromium } from 'playwright';
const b = await chromium.launch();
const p = await b.newPage({ viewport:{width:1280,height:1000} });
await p.goto('file://'+process.cwd()+'/index.html');
for (const id of ['cube','eino','effort']) {
  await p.evaluate(s=>document.getElementById(s).scrollIntoView(), id);
  await p.waitForTimeout(700);
  await p.screenshot({ path:'shot_'+id+'.png' });
}
await b.close(); console.log('shots done');
