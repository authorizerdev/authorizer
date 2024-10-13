'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./reach-alert.cjs.prod.js");
} else {
  module.exports = require("./reach-alert.cjs.dev.js");
}
