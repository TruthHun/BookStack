(function() {

    var factory = function (exports) {

		var pluginName   = "video-dialog";

		exports.fn.videoDialog = function() {

            var _this       = this;
            var cm          = this.cm;
            var lang        = this.lang;
            var editor      = this.editor;
            var settings    = this.settings;
            var cursor      = cm.getCursor();
            var selection   = cm.getSelection();
            var videoLang   = lang.dialog.video;
            var classPrefix = this.classPrefix;
            var iframeName  = classPrefix + "video-iframe";
			var dialogName  = classPrefix + pluginName, dialog;

			cm.focus();

            var loading = function(show) {
                var _loading = dialog.find("." + classPrefix + "dialog-mask");
                _loading[(show) ? "show" : "hide"]();
            };

            if (editor.find("." + dialogName).length < 1)
            {
                var guid   = (new Date).getTime();
                var action = settings.videoUploadURL + (settings.videoUploadURL.indexOf("?") >= 0 ? "&" : "?") + "guid=" + guid+"&type=video";

                if (settings.crossDomainUpload)
                {
                    action += "&callback=" + settings.uploadCallbackURL + "&dialog_id=editormd-video-dialog-" + guid;
                }

                var dialogContent = ( (settings.videoUpload) ? "<form action=\"" + action +"\" target=\"" + iframeName + "\" method=\"post\" enctype=\"multipart/form-data\" class=\"" + classPrefix + "form\">" : "<div class=\"" + classPrefix + "form\">" ) +
                                        ( (settings.videoUpload) ? "<iframe name=\"" + iframeName + "\" id=\"" + iframeName + "\" guid=\"" + guid + "\"></iframe>" : "" ) +
                                        "<label>" + videoLang.url + "</label>" +
                                        "<input type=\"text\" data-url />" + (function(){
                                            return (settings.videoUpload) ? "<div class=\"" + classPrefix + "file-input\">" +
                                                                                "<input type=\"file\" name=\"" + classPrefix + "video-file\" accept=\"video/*\" />" +
                                                                                "<input type=\"submit\" value=\"" + videoLang.uploadButton + "\" />" +
                                                                            "</div>" : "";
                                        })() +
                                        "<br/>" +
                                        "<label>" + videoLang.alt + "</label>" +
                                        "<input type=\"text\" value=\"" + selection + "\" data-alt />" +
                                        "<br/>" +
                                        "<label>" + videoLang.link + "</label>" +
                                        "<input type=\"text\" value=\"\" data-link />" +
                                        "<br/>" +
                                    ( (settings.videoUpload) ? "</form>" : "</div>");

                //var videoFooterHTML = "<button class=\"" + classPrefix + "btn " + classPrefix + "video-manager-btn\" style=\"float:left;\">" + videoLang.managerButton + "</button>";

                dialog = this.createDialog({
                    title      : videoLang.title,
                    width      : (settings.videoUpload) ? 465 : 380,
                    height     : 254,
                    name       : dialogName,
                    content    : dialogContent,
                    mask       : settings.dialogShowMask,
                    drag       : settings.dialogDraggable,
                    lockScreen : settings.dialogLockScreen,
                    maskStyle  : {
                        opacity         : settings.dialogMaskOpacity,
                        backgroundColor : settings.dialogMaskBgColor
                    },
                    buttons : {
                        enter : [lang.buttons.enter, function() {
                            var url  = this.find("[data-url]").val();
                            var alt  = this.find("[data-alt]").val();
                            var link = this.find("[data-link]").val();

                            if (url === "")
                            {
                                alert(videoLang.videoURLEmpty);
                                return false;
                            }
                            cm.replaceSelection('<video controls poster="'+ link +'" src="' + url + '">' + alt + '</video>');
                            // <video controls src=""></video>

							// var altAttr = (alt !== "") ? " \"" + alt + "\"" : "";

                            // if (link === "" || link === "http://")
                            // {
                            //     cm.replaceSelection("![" + alt + "](" + url + altAttr + ")");
                            // }
                            // else
                            // {
                            //     cm.replaceSelection("[![" + alt + "](" + url + altAttr + ")](" + link + altAttr + ")");
                            // }

                            if (alt === "") {
                                cm.setCursor(cursor.line, cursor.ch + 2);
                            }
                            this.hide().lockScreen(false).hideMask();

                            return false;
                        }],

                        cancel : [lang.buttons.cancel, function() {
                            this.hide().lockScreen(false).hideMask();

                            return false;
                        }]
                    }
                });

                dialog.attr("id", classPrefix + "video-dialog-" + guid);

				if (!settings.videoUpload) {
                    return ;
                }

				var fileInput  = dialog.find("[name=\"" + classPrefix + "video-file\"]");

				fileInput.bind("change", function() {
					var fileName  = fileInput.val();
					var isvideo   = new RegExp("(\\.(" + settings.videoFormats.join("|") + "))$"); // /(\.(webp|jpg|jpeg|gif|bmp|png))$/

					if (fileName === "")
					{
						alert(videoLang.uploadFileEmpty);
                        
                        return false;
					}
					
                    if (!isvideo.test(fileName))
					{
						alert(videoLang.formatNotAllowed + settings.videoFormats.join(", "));
                        
                        return false;
					}

                    loading(true);

                    var submitHandler = function() {

                        var uploadIframe = document.getElementById(iframeName);

                        uploadIframe.onload = function() {
                            
                            loading(false);

                            var body = (uploadIframe.contentWindow ? uploadIframe.contentWindow : uploadIframe.contentDocument).document.body;
                            var json = (body.innerText) ? body.innerText : ( (body.textContent) ? body.textContent : null);

                            json = (typeof JSON.parse !== "undefined") ? JSON.parse(json) : eval("(" + json + ")");

                            if (json.success === 1)
                            {
                                dialog.find("[data-url]").val(json.url);
                                dialog.find("[data-alt]").val(json.alt);
                            }
                            else
                            {
                                alert(json.message);
                            }

                            return false;
                        };
                    };

                    dialog.find("[type=\"submit\"]").bind("click", submitHandler).trigger("click");
				});
            }

			dialog = editor.find("." + dialogName);
			dialog.find("[type=\"text\"]").val("");
			dialog.find("[type=\"file\"]").val("");
			dialog.find("[data-link]").val("");

			this.dialogShowMask(dialog);
			this.dialogLockScreen();
			dialog.show();

		};

	};

	// CommonJS/Node.js
	if (typeof require === "function" && typeof exports === "object" && typeof module === "object")
    {
        module.exports = factory;
    }
	else if (typeof define === "function")  // AMD/CMD/Sea.js
    {
		if (define.amd) { // for Require.js

			define(["editormd"], function(editormd) {
                factory(editormd);
            });

		} else { // for Sea.js
			define(function(require) {
                var editormd = require("./../../editormd");
                factory(editormd);
            });
		}
	}
	else
	{
        factory(window.editormd);
	}

})();
