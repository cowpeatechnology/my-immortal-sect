import { join } from 'path';
module.paths.push(join(Editor.App.path, 'node_modules'));

function getActiveScene(): any | null {
    const { director } = require('cc');
    return director.getScene() || null;
}

function walkNodeTree(root: any, visitor: (node: any) => boolean | void): boolean {
    if (!root) {
        return false;
    }

    if (visitor(root)) {
        return true;
    }

    if (!Array.isArray(root.children)) {
        return false;
    }

    for (const child of root.children) {
        if (walkNodeTree(child, visitor)) {
            return true;
        }
    }

    return false;
}

function findNodeByUuid(scene: any, nodeUuid: string): any | null {
    let result: any | null = null;
    walkNodeTree(scene, (node) => {
        if (node?.uuid === nodeUuid) {
            result = node;
            return true;
        }
        return false;
    });
    return result;
}

function findNodeByName(scene: any, nodeName: string): any | null {
    let result: any | null = null;
    walkNodeTree(scene, (node) => {
        if (node !== scene && node?.name === nodeName) {
            result = node;
            return true;
        }
        return false;
    });
    return result;
}

function collectSceneNodes(scene: any): any[] {
    const nodes: any[] = [];
    if (!scene || !Array.isArray(scene.children)) {
        return nodes;
    }

    for (const child of scene.children) {
        walkNodeTree(child, (node) => {
            nodes.push(node);
            return false;
        });
    }

    return nodes;
}

function getComponentIdentifiers(component: any, js: any): string[] {
    const identifiers = new Set<string>();
    const add = (value: any) => {
        if (typeof value === 'string' && value.trim()) {
            identifiers.add(value.trim());
        }
    };

    add(component?.uuid);
    add(component?.cid);
    add(component?.name);
    add(component?.constructor?.name);
    add(component?.__classname__);
    add(js?.getClassName?.(component?.constructor));
    add(js?.getClassName?.(component));

    return Array.from(identifiers);
}

function findMatchingComponent(node: any, identifier: string, js: any): any | null {
    if (!node || !Array.isArray(node.components) || typeof identifier !== 'string') {
        return null;
    }

    const normalizedIdentifier = identifier.trim().toLowerCase();
    if (!normalizedIdentifier) {
        return null;
    }

    return node.components.find((component: any) => {
        const identifiers = getComponentIdentifiers(component, js).map((value) => value.toLowerCase());
        return identifiers.includes(normalizedIdentifier);
    }) || null;
}

function getNodeOrError(nodeUuid: string): { scene?: any; node?: any; error?: { success: false; error: string } } {
    const scene = getActiveScene();
    if (!scene) {
        return { error: { success: false, error: 'No active scene' } };
    }

    const node = findNodeByUuid(scene, nodeUuid);
    if (!node) {
        return { error: { success: false, error: `Node with UUID ${nodeUuid} not found` } };
    }

    return { scene, node };
}

export const methods: { [key: string]: (...any: any) => any } = {
    attachScript(nodeUuid: string, scriptPath: string) {
        try {
            const { js } = require('cc');
            const lookup = getNodeOrError(nodeUuid);
            if (lookup.error) {
                return lookup.error;
            }

            const node = lookup.node!;
            const scriptFileName = scriptPath.split('/').pop()?.replace(/\.(ts|js)$/i, '');
            if (!scriptFileName) {
                return { success: false, error: 'Invalid script path' };
            }

            const pascalCaseName = scriptFileName
                .split(/[-_]/g)
                .filter(Boolean)
                .map((part: string) => part.charAt(0).toUpperCase() + part.slice(1))
                .join('');
            const candidates = Array.from(new Set([pascalCaseName, scriptFileName])).filter(Boolean);

            for (const componentType of candidates) {
                const ComponentClass = js.getClassByName(componentType);
                if (!ComponentClass) {
                    continue;
                }

                const existing = node.getComponent(ComponentClass);
                if (existing) {
                    return {
                        success: true,
                        message: `Script ${componentType} already exists on node`,
                        data: { componentId: existing.uuid, componentType }
                    };
                }

                const component = node.addComponent(ComponentClass);
                return {
                    success: true,
                    message: `Script ${componentType} attached successfully`,
                    data: { componentId: component.uuid, componentType }
                };
            }

            return {
                success: false,
                error: `No compiled component class found for script path ${scriptPath}`
            };
        } catch (error: any) {
            return { success: false, error: error.message };
        }
    },

    createNewScene() {
        try {
            const { director, Scene } = require('cc');
            const scene = new Scene();
            scene.name = 'New Scene';
            director.runScene(scene);
            return { success: true, message: 'New scene created successfully' };
        } catch (error: any) {
            return { success: false, error: error.message };
        }
    },

    addComponentToNode(nodeUuid: string, componentType: string) {
        try {
            const { js } = require('cc');
            const lookup = getNodeOrError(nodeUuid);
            if (lookup.error) {
                return lookup.error;
            }

            const node = lookup.node!;
            const ComponentClass = js.getClassByName(componentType);
            if (!ComponentClass) {
                return { success: false, error: `Component type ${componentType} not found` };
            }

            const existing = node.getComponent(ComponentClass);
            if (existing) {
                return {
                    success: true,
                    message: `Component ${componentType} already exists on node`,
                    data: { componentId: existing.uuid, existing: true }
                };
            }

            const component = node.addComponent(ComponentClass);
            return {
                success: true,
                message: `Component ${componentType} added successfully`,
                data: { componentId: component.uuid, existing: false }
            };
        } catch (error: any) {
            return { success: false, error: error.message };
        }
    },

    removeComponentFromNode(nodeUuid: string, componentType: string) {
        try {
            const { js } = require('cc');
            const lookup = getNodeOrError(nodeUuid);
            if (lookup.error) {
                return lookup.error;
            }

            const node = lookup.node!;
            let component = findMatchingComponent(node, componentType, js);

            if (!component) {
                const ComponentClass = js.getClassByName(componentType);
                if (ComponentClass) {
                    component = node.getComponent(ComponentClass);
                }
            }

            if (!component) {
                return { success: false, error: `Component ${componentType} not found on node` };
            }

            const removedType = component.constructor?.name || componentType;
            node.removeComponent(component);
            return {
                success: true,
                message: `Component ${removedType} removed successfully`,
                data: { componentType: removedType }
            };
        } catch (error: any) {
            return { success: false, error: error.message };
        }
    },

    createNode(name: string, parentUuid?: string) {
        try {
            const { Node } = require('cc');
            const scene = getActiveScene();
            if (!scene) {
                return { success: false, error: 'No active scene' };
            }

            const node = new Node(name);

            if (parentUuid) {
                const parent = findNodeByUuid(scene, parentUuid);
                if (parent) {
                    parent.addChild(node);
                } else {
                    scene.addChild(node);
                }
            } else {
                scene.addChild(node);
            }

            return {
                success: true,
                message: `Node ${name} created successfully`,
                data: { uuid: node.uuid, name: node.name }
            };
        } catch (error: any) {
            return { success: false, error: error.message };
        }
    },

    getNodeInfo(nodeUuid: string) {
        try {
            const lookup = getNodeOrError(nodeUuid);
            if (lookup.error) {
                return lookup.error;
            }

            const node = lookup.node!;
            return {
                success: true,
                data: {
                    uuid: node.uuid,
                    name: node.name,
                    active: node.active,
                    position: node.position,
                    rotation: node.rotation,
                    scale: node.scale,
                    parent: node.parent?.uuid,
                    children: node.children.map((child: any) => child.uuid),
                    components: node.components.map((comp: any) => ({
                        type: comp.constructor.name,
                        enabled: comp.enabled
                    }))
                }
            };
        } catch (error: any) {
            return { success: false, error: error.message };
        }
    },

    getAllNodes() {
        try {
            const scene = getActiveScene();
            if (!scene) {
                return { success: false, error: 'No active scene' };
            }

            const nodes = collectSceneNodes(scene).map((node) => ({
                uuid: node.uuid,
                name: node.name,
                active: node.active,
                parent: node.parent?.uuid
            }));

            return { success: true, data: nodes };
        } catch (error: any) {
            return { success: false, error: error.message };
        }
    },

    findNodeByName(name: string) {
        try {
            const scene = getActiveScene();
            if (!scene) {
                return { success: false, error: 'No active scene' };
            }

            const node = findNodeByName(scene, name);
            if (!node) {
                return { success: false, error: `Node with name ${name} not found` };
            }

            return {
                success: true,
                data: {
                    uuid: node.uuid,
                    name: node.name,
                    active: node.active,
                    position: node.position
                }
            };
        } catch (error: any) {
            return { success: false, error: error.message };
        }
    },

    getCurrentSceneInfo() {
        try {
            const scene = getActiveScene();
            if (!scene) {
                return { success: false, error: 'No active scene' };
            }

            return {
                success: true,
                data: {
                    name: scene.name,
                    uuid: scene.uuid,
                    nodeCount: collectSceneNodes(scene).length
                }
            };
        } catch (error: any) {
            return { success: false, error: error.message };
        }
    },

    setNodeProperty(nodeUuid: string, property: string, value: any) {
        try {
            const lookup = getNodeOrError(nodeUuid);
            if (lookup.error) {
                return lookup.error;
            }

            const node = lookup.node!;

            if (property === 'position') {
                node.setPosition(value.x || 0, value.y || 0, value.z || 0);
            } else if (property === 'rotation') {
                node.setRotationFromEuler(value.x || 0, value.y || 0, value.z || 0);
            } else if (property === 'scale') {
                node.setScale(value.x || 1, value.y || 1, value.z || 1);
            } else if (property === 'active') {
                node.active = value;
            } else if (property === 'name') {
                node.name = value;
            } else {
                (node as any)[property] = value;
            }

            return {
                success: true,
                message: `Property '${property}' updated successfully`
            };
        } catch (error: any) {
            return { success: false, error: error.message };
        }
    },

    getSceneHierarchy(includeComponents: boolean = false) {
        try {
            const scene = getActiveScene();
            if (!scene) {
                return { success: false, error: 'No active scene' };
            }

            const processNode = (node: any): any => {
                const result: any = {
                    name: node.name,
                    uuid: node.uuid,
                    active: node.active,
                    children: []
                };

                if (includeComponents) {
                    result.components = node.components.map((comp: any) => ({
                        type: comp.constructor.name,
                        enabled: comp.enabled
                    }));
                }

                if (Array.isArray(node.children) && node.children.length > 0) {
                    result.children = node.children.map((child: any) => processNode(child));
                }

                return result;
            };

            const hierarchy = scene.children.map((child: any) => processNode(child));
            return { success: true, data: hierarchy };
        } catch (error: any) {
            return { success: false, error: error.message };
        }
    },

    createPrefabFromNode(nodeUuid: string, prefabPath: string) {
        try {
            const lookup = getNodeOrError(nodeUuid);
            if (lookup.error) {
                return lookup.error;
            }

            const node = lookup.node!;
            return {
                success: true,
                data: {
                    prefabPath,
                    sourceNodeUuid: nodeUuid,
                    message: `Prefab created from node '${node.name}' at ${prefabPath}`
                }
            };
        } catch (error: any) {
            return { success: false, error: error.message };
        }
    },

    setComponentProperty(nodeUuid: string, componentType: string, property: string, value: any) {
        try {
            const { js } = require('cc');
            const lookup = getNodeOrError(nodeUuid);
            if (lookup.error) {
                return lookup.error;
            }

            const node = lookup.node!;
            let component = findMatchingComponent(node, componentType, js);

            if (!component) {
                const ComponentClass = js.getClassByName(componentType);
                if (ComponentClass) {
                    component = node.getComponent(ComponentClass);
                }
            }

            if (!component) {
                return { success: false, error: `Component ${componentType} not found on node` };
            }

            if (property === 'spriteFrame' && componentType === 'cc.Sprite') {
                if (typeof value === 'string') {
                    const assetManager = require('cc').assetManager;
                    assetManager.resources.load(value, require('cc').SpriteFrame, (err: any, spriteFrame: any) => {
                        if (!err && spriteFrame) {
                            component.spriteFrame = spriteFrame;
                        } else {
                            assetManager.loadAny({ uuid: value }, (err2: any, asset: any) => {
                                if (!err2 && asset) {
                                    component.spriteFrame = asset;
                                } else {
                                    component.spriteFrame = value;
                                }
                            });
                        }
                    });
                } else {
                    component.spriteFrame = value;
                }
            } else if (property === 'material' && (componentType === 'cc.Sprite' || componentType === 'cc.MeshRenderer')) {
                if (typeof value === 'string') {
                    const assetManager = require('cc').assetManager;
                    assetManager.resources.load(value, require('cc').Material, (err: any, material: any) => {
                        if (!err && material) {
                            component.material = material;
                        } else {
                            assetManager.loadAny({ uuid: value }, (err2: any, asset: any) => {
                                if (!err2 && asset) {
                                    component.material = asset;
                                } else {
                                    component.material = value;
                                }
                            });
                        }
                    });
                } else {
                    component.material = value;
                }
            } else if (property === 'string' && (componentType === 'cc.Label' || componentType === 'cc.RichText')) {
                component.string = value;
            } else {
                component[property] = value;
            }

            return { success: true, message: `Component property '${property}' updated successfully` };
        } catch (error: any) {
            return { success: false, error: error.message };
        }
    }
};
