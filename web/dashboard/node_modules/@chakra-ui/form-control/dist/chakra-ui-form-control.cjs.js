'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-form-control.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-form-control.cjs.dev.js");
}
