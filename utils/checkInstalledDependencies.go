package utils

import (
	"fmt"
	"io/ioutil"
	"os"
)

const testPuppeteerJS = `'use strict';
const puppeteer = require('puppeteer');
async function test() {
    const browser = await puppeteer.launch({args: ['--no-sandbox', '--disable-setuid-sandbox'], headless: true, ignoreHTTPSErrors: true});
    const page = await browser.newPage();
    await page.goto('http://localhost:%v', {"waitUntil" :  ['networkidle2', 'domcontentloaded'], "timeout": 10000});
    let content=await page.content();
    console.log(content);
    await browser.close();
}
test();`

// IsInstalledPuppetter 是否安装了puppeteer
func IsInstalledPuppetter(listenPort int) (err error) {
	testFile := "test.js"
	defer func() {
		os.Remove(testFile)
	}()
	ioutil.WriteFile(testFile, []byte(fmt.Sprintf(testPuppeteerJS, listenPort)), os.ModePerm)
	_, err = ExecCommand("node", []string{testFile})
	return
}

// IsInstalledGit 是否安装了Git
func IsInstalledGit() (err error) {
	_, err = ExecCommand("git", []string{"--version"})
	return
}

// IsInstalledChrome 是否安装了Chrome
func IsInstalledChrome(chrome string) (err error) {
	_, err = ExecCommand(chrome, []string{"--version"})
	return
}

// IsInstalledCalibre 是否安装了calibre
func IsInstalledCalibre(calibre string) (err error) {
	_, err = ExecCommand(calibre, []string{"--version"})
	return
}
