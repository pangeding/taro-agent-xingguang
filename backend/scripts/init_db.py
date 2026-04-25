#!/usr/bin/env python3
"""
数据库初始化脚本
用于创建数据库表和导入基础数据
"""
import sys
import os

sys.path.append(os.path.join(os.path.dirname(__file__), ".."))

from app.db.models import create_tables
from app.db.base import db


def init_database():
    """初始化数据库"""
    print("正在初始化数据库...")

    # 创建所有表
    with db:
        create_tables()

    print("数据库表创建完成！")


if __name__ == "__main__":
    init_database()