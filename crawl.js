'use strict';

const puppeteer = require('puppeteer');
const fs = require("fs");

let args = process.argv.splice(2);
let l=args.length;
let url, folder, selector;

for(let i=0;i<l;i++){
    switch (args[i]){
        case "--url":
            url = args[i+1];
            if (url==undefined){
                url = "";
            }
            break;
        case '--folder':
            folder = args[i+1];
            break;
        case '--selector':
            selector = args.splice(i+1).join(" ");
            i=l;
            break;
    }
    i++;
}

async function screenshot() {
    const browser = await puppeteer.launch({args: ['--no-sandbox', '--disable-setuid-sandbox'], headless: true, ignoreHTTPSErrors: true});
    const page = await browser.newPage();
    let shot = false;
    if(folder && selector){
        shot = true;
        page.setViewport({width: 1280, height: 20480});
    }

    page.setExtraHTTPHeaders({
        "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8,co;q=0.7,fr;q=0.6,zh-HK;q=0.5,zh-TW;q=0.4",
        "User-Agent": "Mozi lla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3766.2 Safari/537.36"
    });

    await page.setRequestInterception(true);
    page.on("request", request => {
        getHost(request.url()).indexOf("google")>-1 ? request.abort() : request.continue();
    });
    let timeout =  shot ? 120000 : 60000;

    await page.goto(url, {"waitUntil" :  ['networkidle2', 'domcontentloaded'], "timeout":timeout});
    let res;
    if(shot){
        if (folder.substr(folder.length-1,1)!="/"){
            folder=folder+"/"
        }
        res = await page.evaluate(function (ele) {
            let bodyHeight = document.querySelector("body").clientHeight

            let data = new Array();
            let eleSlice=ele.split(",");

            for (let i = 0; i < eleSlice.length; i++) {
                let d = [],item = eleSlice[i];
                let elements = document.querySelectorAll(item);
                for (var element of elements){
                    let bounding = element.getBoundingClientRect();
                    let x = bounding.x;
                    let y = bounding.y;
                    let width = bounding.width;
                    let height = bounding.height;
                    d.push({x, y, width, height});
                }
                data.push(d)
            }

            return {height: bodyHeight, data: data};
        }, selector);
        fs.writeFile(folder+'screenshot.json', JSON.stringify(res),function(){});
        await page.screenshot({path: folder+'screenshot.png', fullPage: true});
    }
    let content=await page.content();
    console.log(content);
    await browser.close();
}

function getHost(url) {
    let u = String(url).toLowerCase()
    if (u.startsWith("https://") || u.startsWith("http://")){
        return u.split("/")[2]
    }
    return ""
}
if (url) screenshot();