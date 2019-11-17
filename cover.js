'use strict';
// 说明： 这段js是用来进行书籍封面截屏的，请勿随便删除或改动
const puppeteer = require('puppeteer');
const fs = require("fs");

let args = process.argv.splice(2);
let l=args.length;
let url, identify,folder;

for(let i=0;i<l;i++){
    switch (args[i]){
        case "--url":
            url = args[i+1];
            if (url==undefined){
                url = "";
            }
            break;
        case '--identify':
            identify = args[i+1];
            break;
    }
    i++;
}

async function screenshot() {
    const browser = await puppeteer.launch({args: ['--no-sandbox', '--disable-setuid-sandbox'], headless: true, ignoreHTTPSErrors: true});
    const page = await browser.newPage();
    page.setViewport({width: 800, height: 1068}); // A4 宽高
    folder="cache/books/"+identify+"/"
    page.setExtraHTTPHeaders({
        "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8,co;q=0.7,fr;q=0.6,zh-HK;q=0.5,zh-TW;q=0.4",
        "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3766.2 Safari/537.36"
    });

    await page.goto(url, {"waitUntil" :  ['networkidle2', 'domcontentloaded'], "timeout":5000});
    await page.screenshot({path: folder+'cover.png'});
    await browser.close();
}
if (url && identify) screenshot();