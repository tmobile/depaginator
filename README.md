# Depaginator Paginated API Iterator

[![Tag](https://img.shields.io/github/tag/tmobile/depaginator.svg)](https://github.com/tmobile/depaginator/tags)
[![License](https://img.shields.io/hexpm/l/plug.svg)](https://github.com/tmobile/depaginator/blob/main/LICENSE)
[![Godoc](https://pkg.go.dev/badge/github.com/tmobile/depaginator)](https://pkg.go.dev/github.com/tmobile/depaginator)
[![Issue Tracker](https://img.shields.io/github/issues/tmobile/depaginator.svg)](https://github.com/tmobile/depaginator/issues)
[![Pull Request Tracker](https://img.shields.io/github/issues-pr/tmobile/depaginator.svg)](https://github.com/tmobile/depaginator/pulls)
[![Report Card](https://goreportcard.com/badge/github.com/tmobile/depaginator)](https://goreportcard.com/report/github.com/tmobile/depaginator)

This repository contains the Depaginator.  The Depaginator is a tool for traversing all items presented by a paginated API: all pages are retrieved by independent goroutines, then each item in each page is iterated over, calling a `Handle` method.  The index of the item in the list is also passed to `Handle` for the benefit of ordering-sensitive applications.

## How to Use

For full details, refer to the [package documentation](https://pkg.go.dev/github.com/tmobile/depaginator).  The basic concept is for the consuming application to create one object that implements a `GetPage`, which conforms to the `PageGetter` interface, and a second object that implements `Handle`, conforming to the `Handler` interface.  The `GetPage` method is passed a `PageRequest`, which bundles a `PageIndex` integer with an application-defined `Request`.  The `GetPage` method must then retrieve the desired page of the results, add any relevant metadata--including requests for subsequent pages--via calls to the `Depaginator` object, and return a an array of items.  The `Depaginator` will then call `Handle` for each element in the returned list.  Optionally, the `Handler` may implement additional `Start`, `Update`, or `Done` methods which will be called at appropriate parts of the workflow.

To actually perform the depagination operation, the application passes instances of these objects and any appropriate options to the `Depaginate` function; this returns a `Depaginator` object which the application may then `Wait` on.  Any errors encountered during the operation will be returned by `Wait`.

The `PageGetter` and the `Handler` interfaces are distinct to aid in code reuse; this architecture allows for general handlers like the provided `ListHandler`, as well as allowing the `PageGetter` to be reused with different handlers, depending on the needs of the application.

For convenience, the `ListHandler` type is provided; this is a `Handler` implementation which assembles the list of retrieved items into the correct order.

## Why to Use

Many server APIs that return lists of objects will "paginate" the response to avoid overwhelming the connection or the client--or the server or database.  However, many clients consuming that API need to perform some operation on all the returned objects, such as displaying them to the user or applying additional filters to select specific items.  Especially for large lists, this process can be quite slow; this may be fine for user-interactive clients, such as command line clients, but if some other operation is being performed, such as bulk modifications, this can be unacceptably slow.  The `Depaginator` is intended to simplify the implementation of code that iterates over all the items in a list by allowing their retrieval as fast as the server API will permit.
