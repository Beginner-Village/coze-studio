/*
 * Copyright 2025 ynet-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * Ynet Web SDK v1.0.0
 * Lightweight SDK for embedding Ynet AI chat widget into external websites.
 * Creates an iframe pointing to the Ynet platform and manages communication via postMessage.
 */
(function (root) {
  'use strict';

  var SDK_VERSION = '1.0.0';
  var MESSAGE_SOURCE_HOST = 'ynet-sdk-host';
  var MESSAGE_SOURCE_IFRAME = 'ynet-sdk-iframe';

  /**
   * Messenger - handles postMessage communication between host and iframe
   */
  function Messenger(options) {
    this.options = options;
    this.listeners = {};
    this._boundHandler = this._handleMessage.bind(this);
    window.addEventListener('message', this._boundHandler);
  }

  Messenger.prototype.send = function (type, payload) {
    var target = this.options.targetWindow;
    if (typeof target === 'function') target = target();
    if (!target) return;
    target.postMessage(
      { source: this.options.selfSource, type: type, payload: payload },
      this.options.targetOrigin
    );
  };

  Messenger.prototype.on = function (type, callback) {
    if (!this.listeners[type]) this.listeners[type] = [];
    this.listeners[type].push(callback);
  };

  Messenger.prototype.destroy = function () {
    window.removeEventListener('message', this._boundHandler);
    this.listeners = {};
  };

  Messenger.prototype._handleMessage = function (event) {
    var origin = this.options.targetOrigin;
    if (origin !== '*' && event.origin !== origin) return;
    var data = event.data;
    if (!data || !data.type || data.source === this.options.selfSource) return;
    var cbs = this.listeners[data.type];
    if (cbs) {
      for (var i = 0; i < cbs.length; i++) {
        cbs[i](data.payload, data);
      }
    }
  };

  /**
   * WebChatClient - main SDK class
   *
   * Usage:
   *   new CozeWebSDK.WebChatClient({
   *     config: {
   *       bot_id: '123456',
   *     },
   *     componentProps: {
   *       title: 'AI Assistant',
   *       width: 400,
   *       height: 600,
   *     },
   *     auth: {
   *       type: 'token',
   *       token: 'pat_xxx',
   *       onRefreshToken: function() { return 'new_token'; }
   *     },
   *     el: '#chat-container',      // optional, mount to specific element
   *     onReady: function() {},      // optional, called when chat is ready
   *     onError: function(err) {},   // optional, called on error
   *   });
   */
  function WebChatClient(options) {
    if (!options) throw new Error('WebChatClient: options is required');

    this.options = options;
    this.iframe = null;
    this.messenger = null;
    this._destroyed = false;

    this._init();
  }

  WebChatClient.prototype._init = function () {
    var config = this.options.config || {};
    var auth = this.options.auth || {};
    var componentProps = this.options.componentProps || {};

    // Determine the base URL for the iframe
    var baseUrl = this.options.baseUrl || this._detectBaseUrl();

    // Build iframe URL with query params
    var botId = config.bot_id || config.appId || '';
    var token = auth.token || '';
    var iframeUrl =
      baseUrl +
      '/agent-chat?bot_id=' +
      encodeURIComponent(botId) +
      '&token=' +
      encodeURIComponent(token) +
      '&mode=websdk' +
      '&parent_origin=' + encodeURIComponent(window.location.origin);

    if (config.workflowId) {
      iframeUrl += '&workflow_id=' + encodeURIComponent(config.workflowId);
    }

    // Create iframe
    this.iframe = document.createElement('iframe');
    this.iframe.src = iframeUrl;
    this.iframe.style.width = (componentProps.width || '100%') + (typeof componentProps.width === 'number' ? 'px' : '');
    this.iframe.style.height = (componentProps.height || '100%') + (typeof componentProps.height === 'number' ? 'px' : '');
    this.iframe.style.border = 'none';
    this.iframe.style.borderRadius = componentProps.borderRadius || '8px';
    this.iframe.style.overflow = 'hidden';
    this.iframe.setAttribute('allow', 'clipboard-write; microphone');

    // Mount to container
    var container = null;
    if (this.options.el) {
      container =
        typeof this.options.el === 'string'
          ? document.querySelector(this.options.el)
          : this.options.el;
    }
    if (!container) {
      // Create a floating container if no el specified
      container = this._createFloatingContainer(componentProps);
    }
    container.appendChild(this.iframe);

    // Setup messenger
    var self = this;
    this.messenger = new Messenger({
      targetWindow: function () {
        return self.iframe ? self.iframe.contentWindow : null;
      },
      targetOrigin: baseUrl || '*',
      selfSource: MESSAGE_SOURCE_HOST,
    });

    this.messenger.on('READY', function () {
      // Send initial config
      self.messenger.send('INIT', {
        bot_id: botId,
        token: token,
        title: componentProps.title,
      });
      if (self.options.onReady) self.options.onReady();
    });

    this.messenger.on('TOKEN_EXPIRED', function () {
      if (auth.onRefreshToken) {
        var result = auth.onRefreshToken();
        if (result && typeof result.then === 'function') {
          result.then(function (newToken) {
            self.updateToken(newToken);
          });
        } else if (result) {
          self.updateToken(result);
        }
      }
    });

    this.messenger.on('ERROR', function (err) {
      if (self.options.onError) self.options.onError(err);
    });
  };

  WebChatClient.prototype._detectBaseUrl = function () {
    // Try to detect from the SDK script tag's src
    var scripts = document.querySelectorAll('script[src]');
    for (var i = 0; i < scripts.length; i++) {
      var src = scripts[i].src;
      if (src.indexOf('ynet-web-sdk') !== -1) {
        var url = new URL(src);
        return url.origin;
      }
    }
    return '';
  };

  WebChatClient.prototype._createFloatingContainer = function (props) {
    var container = document.createElement('div');
    container.style.position = 'fixed';
    container.style.bottom = (props.bottom || 20) + 'px';
    container.style.right = (props.right || 20) + 'px';
    container.style.width = (props.width || 400) + 'px';
    container.style.height = (props.height || 600) + 'px';
    container.style.zIndex = props.zIndex || '9999';
    container.style.boxShadow = '0 4px 24px rgba(0, 0, 0, 0.15)';
    container.style.borderRadius = props.borderRadius || '8px';
    container.style.overflow = 'hidden';
    document.body.appendChild(container);
    this._floatingContainer = container;
    return container;
  };

  WebChatClient.prototype.updateToken = function (token) {
    if (this.messenger) {
      this.messenger.send('UPDATE_TOKEN', { token: token });
    }
  };

  WebChatClient.prototype.updateConfig = function (config) {
    if (this.messenger) {
      this.messenger.send('UPDATE_CONFIG', config);
    }
  };

  WebChatClient.prototype.destroy = function () {
    if (this._destroyed) return;
    this._destroyed = true;
    if (this.messenger) this.messenger.destroy();
    if (this.iframe && this.iframe.parentNode) {
      this.iframe.parentNode.removeChild(this.iframe);
    }
    if (this._floatingContainer && this._floatingContainer.parentNode) {
      this._floatingContainer.parentNode.removeChild(this._floatingContainer);
    }
    this.iframe = null;
    this.messenger = null;
  };

  // Export
  var CozeWebSDK = {
    WebChatClient: WebChatClient,
    version: SDK_VERSION,
  };

  if (typeof module !== 'undefined' && module.exports) {
    module.exports = CozeWebSDK;
  } else if (typeof define === 'function' && define.amd) {
    define(function () {
      return CozeWebSDK;
    });
  }

  root.CozeWebSDK = CozeWebSDK;
})(typeof globalThis !== 'undefined' ? globalThis : typeof self !== 'undefined' ? self : this);
