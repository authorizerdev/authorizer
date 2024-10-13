'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./reach-visually-hidden.cjs.prod.js");
} else {
  module.exports = require("./reach-visually-hidden.cjs.dev.js");
}
