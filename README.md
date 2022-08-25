# BBBBB (BitBlockBoom Bulletin Board)

Welcome to a fun workshop with Voltage! In this workshop we are going to enhance a bulletin board! Currently the bulletin board will accept any comments via an API POST request. However this leads to lots of spam.

To make this better, we're going to only let the comments be made via Keysend and we're also going to make sure the comment is nice before accepting any payment.

## Structure

The `start` folder is the bulletin board without any Lightning integration.
The `result` folder is the bulletin board _with_ Lightning integration.