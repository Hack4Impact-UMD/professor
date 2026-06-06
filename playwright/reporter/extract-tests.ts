import * as acorn from 'acorn';
import { tsPlugin } from 'acorn-typescript';
import * as fs from 'fs';
import * as path from 'path';

interface TestSuite {
  name: string;
  tests: string[];
}

interface TestRepo {
  suites: TestSuite[];
}

const TSParser = acorn.Parser.extend(tsPlugin() as unknown as Parameters<typeof acorn.Parser.extend>[0]);

function findSpecFiles(dir: string): string[] {
  const results: string[] = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === 'node_modules') continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      results.push(...findSpecFiles(full));
    } else if (entry.isFile() && entry.name.endsWith('.spec.ts')) {
      results.push(full);
    }
  }
  return results;
}

function isDescribeCall(node: acorn.Node): node is acorn.CallExpression {
  if (node.type !== 'CallExpression') return false;
  const call = node as acorn.CallExpression;
  const callee = call.callee;
  if (callee.type === 'Identifier') {
    return (callee as acorn.Identifier).name === 'describe';
  }
  if (callee.type === 'MemberExpression') {
    const prop = (callee as acorn.MemberExpression).property;
    return prop.type === 'Identifier' && (prop as acorn.Identifier).name === 'describe';
  }
  return false;
}

function isTestCall(node: acorn.Node): node is acorn.CallExpression {
  if (node.type !== 'CallExpression') return false;
  const call = node as acorn.CallExpression;
  return (
    call.callee.type === 'Identifier' &&
    (call.callee as acorn.Identifier).name === 'test'
  );
}

function getFirstStringArg(callExpr: acorn.CallExpression): string | null {
  const arg = callExpr.arguments[0];
  if (!arg || arg.type !== 'Literal') return null;
  const value = (arg as acorn.Literal).value;
  return typeof value === 'string' ? value : null;
}

function getCallbackBody(callExpr: acorn.CallExpression): acorn.Statement[] | null {
  const last = callExpr.arguments[callExpr.arguments.length - 1];
  if (!last || (last.type !== 'ArrowFunctionExpression' && last.type !== 'FunctionExpression')) {
    return null;
  }
  const fn = last as acorn.ArrowFunctionExpression | acorn.FunctionExpression;
  if (!fn.body || fn.body.type !== 'BlockStatement') return null;
  return (fn.body as acorn.BlockStatement).body;
}

function extractFromFile(source: string): TestSuite[] {
  const ast = TSParser.parse(source, {
    ecmaVersion: 'latest',
    sourceType: 'module',
  }) as acorn.Program;

  const suites: TestSuite[] = [];

  for (const stmt of ast.body) {
    if (stmt.type !== 'ExpressionStatement') continue;
    const expr = (stmt as acorn.ExpressionStatement).expression;
    if (!isDescribeCall(expr)) continue;

    const name = getFirstStringArg(expr);
    if (name === null) continue;

    const body = getCallbackBody(expr);
    if (body === null) continue;

    const tests: string[] = [];
    for (const inner of body) {
      if (inner.type !== 'ExpressionStatement') continue;
      const innerExpr = (inner as acorn.ExpressionStatement).expression;
      if (!isTestCall(innerExpr)) continue;
      const testName = getFirstStringArg(innerExpr);
      if (testName !== null) tests.push(testName);
    }

    suites.push({ name, tests });
  }

  return suites;
}

function main(): void {
  const rootDir = path.resolve(process.argv[2] ?? '.');
  const specFiles = findSpecFiles(rootDir);

  const allSuites: TestSuite[] = [];
  for (const file of specFiles) {
    const source = fs.readFileSync(file, 'utf-8');
    try {
      allSuites.push(...extractFromFile(source));
    } catch (err) {
      process.stderr.write(`Warning: failed to parse ${file}: ${err}\n`);
    }
  }

  const output: TestRepo = { suites: allSuites };
  process.stdout.write(JSON.stringify(output, null, 2) + '\n');
}

main();
