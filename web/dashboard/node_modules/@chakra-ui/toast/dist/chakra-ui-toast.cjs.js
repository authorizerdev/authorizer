'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-toast.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-toast.cjs.dev.js");
}
