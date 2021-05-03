# Depaginator Paginated API Iterator

[![Tag](https://img.shields.io/github/tag/tmobile/depaginator.svg)](https://github.com/tmobile/depaginator/tags)
[![License](https://img.shields.io/hexpm/l/plug.svg)](https://github.com/tmobile/depaginator/blob/main/LICENSE)
[![Test Report](https://travis-ci.com/tmobile/depaginator.svg?branch=main)](https://travis-ci.com/tmobile/depaginator)
[![Godoc](https://pkg.go.dev/badge/github.com/tmobile/depaginator)](https://pkg.go.dev/github.com/tmobile/depaginator)
[![Issue Tracker](https://img.shields.io/github/issues/tmobile/depaginator.svg)](https://github.com/tmobile/depaginator/issues)
[![Pull Request Tracker](https://img.shields.io/github/issues-pr/tmobile/depaginator.svg)](https://github.com/tmobile/depaginator/pulls)
[![Report Card](https://goreportcard.com/badge/github.com/tmobile/depaginator)](https://goreportcard.com/report/github.com/tmobile/depaginator)

This repository contains the Depaginator.  The Depaginator is a tool for traversing all items presented by a paginated API; it iterates over every item, calling a `HandleItem` method.  It does this iteration using goroutines, meaning there is no definite order in which items are passed to `HandleItem`; but the index of the item in the response is also passed to `HandleItem` for the benefit of ordering-sensitive applications.  Requests for pages of items are also handled with goroutines, allowing all items to be handled in the shortest possible time permitted by any rate limits defined on the API.

## How to Use

For full details, refer to the [documentation](https://pkg.go.dev/github.com/tmobile/depaginator).  The basic concept is for the consuming application to create an object that implements `GetPage`, `HandleItem`, and `Done` methods, conforming to the `depaginator.API` interface.  The `GetPage` method is passed a `PageMeta` object and a `PageRequest`, which bundles a `PageIndex` integer with an application-defined `Request`.  The `GetPage` method must then retrieve the desired page of the results, add any relevant metadata--including requests for subsequent pages--to the `PageMeta`, and return a `Page` (another interface implementing `Len` and `Get` methods).  The `Depaginator` will then call `HandleItem` for each element in the `Page` (in an independent goroutine).  After all pages have been retrieved and all items handled, the consuming application calls `Depaginator.Wait`, which will call the `Done` method (passing it the final `PageMeta`) and return a list of any page retrieval errors that were encountered.

## Why to Use

Many server APIs that return lists of objects will "paginate" the response to avoid overwhelming the connection or the client--or the server or database.  However, many clients consuming that API need to perform some operation on all the returned objects, such as displaying them to the user or applying additional filters to select specific items.  Especially for large lists, this process can be quite slow; this may be fine for user-interactive clients, such as command line clients, but if some other operation is being performed, such as bulk modifications, this can be unacceptably slow.  The `Depaginator` is intended to simplify the implementation of code that iterates over all the items in a list by allowing their retrieval as fast as the server API will permit.
