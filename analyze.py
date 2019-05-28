import sys
import argparse
import json


def cnt_on_rule_id(issues, rule_id):
    return len([issue for issue in issues if issue['rule_id'] == rule_id])

def analyze(js):
    issues = js['Issues']
    if not issues:
        print "Security check: no security issue detected"
        return 0

    for issue in issues:
        f = issue['file']
        f = '/'.join(f.split('/')[2:])
        issue['file'] = f

    must_fix = []
    better_fix = []
    for issue in issues:
        if issue['severity'] == 'HIGH':
            must_fix.append(issue)
        else:
            better_fix.append(issue)


    print '======== Must fix the potential security issues ========'
    for issue in must_fix:
        print json.dumps(issue, indent=4)

    print '======== Optional to fix the potential security issues ========'
    for issue in better_fix:
        print json.dumps(issue, indent=4)

    if must_fix:
        return 1
    else:
        return 0


def parse_args_or_exit(argv=None):
    """
    Parse command line options
    """
    parser = argparse.ArgumentParser(description="Analyze security check result")
    parser.add_argument("check_result", metavar="check_result",
            help="json file of check result")

    args = parser.parse_args(argv)

    return args

def main(argv):
    args = parse_args_or_exit(argv)
    check_result = args.check_result
    with open(args.check_result) as f:
        js = json.load(f)
        sys.exit(analyze(js))

if __name__ == '__main__':
    main(sys.argv[1:])
